// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsscraper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/liatrio/liatrio-otel-collector/receiver/azuredevopsreceiver/internal/metadata"
)

var errClientNotInitErr = errors.New("http client not initialized")

type azuredevopsScraper struct {
	client   *http.Client
	cfg      *Config
	settings component.TelemetrySettings
	logger   *zap.Logger
	mb       *metadata.MetricsBuilder
	rb       *metadata.ResourceBuilder
}

func (ados *azuredevopsScraper) start(ctx context.Context, host component.Host) (err error) {
	ados.logger.Sugar().Info("Starting the Azure DevOps scraper")
	ados.client, err = ados.cfg.ToClient(ctx, host, ados.settings)
	return
}

func newAzureDevOpsScraper(
	_ context.Context,
	settings receiver.Settings,
	cfg *Config,
) *azuredevopsScraper {
	return &azuredevopsScraper{
		cfg:      cfg,
		settings: settings.TelemetrySettings,
		logger:   settings.Logger,
		mb:       metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings),
		rb:       metadata.NewResourceBuilder(cfg.ResourceAttributes),
	}
}

// Scrape the Azure DevOps REST API for various metrics
func (ados *azuredevopsScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	ados.logger.Sugar().Debug("checking if client is initialized")
	if ados.client == nil {
		return pmetric.NewMetrics(), errClientNotInitErr
	}

	now := pcommon.NewTimestampFromTime(time.Now())
	ados.logger.Sugar().Debugf("current time: %v", now)

	ados.logger.Sugar().Debug("creating Azure DevOps REST API client")

	// Get repositories from the specified project
	repos, err := ados.getRepositories(ctx, ados.cfg.Project)
	if err != nil {
		ados.logger.Sugar().Errorf("error getting repositories for project '%s': %v", ados.cfg.Project, err)
		return ados.mb.Emit(), err
	}

	// Record repository count metric
	ados.mb.RecordVcsRepositoryCountDataPoint(now, int64(len(repos)))

	var wg sync.WaitGroup
	var mux sync.Mutex

	var max int
	switch {
	case ados.cfg.ConcurrencyLimit > 0:
		max = ados.cfg.ConcurrencyLimit
	default:
		max = len(repos) + 1
	}

	limiter := make(chan struct{}, max)

	// Process each repository
	for _, repo := range repos {
		wg.Add(1)
		repo := repo

		limiter <- struct{}{}

		go func() {
			defer func() {
				<-limiter
				wg.Done()
			}()

			// Get branches for this repository
			branches, err := ados.getBranches(ctx, ados.cfg.Project, repo.ID)
			if err != nil {
				ados.logger.Sugar().Errorf("error getting branches for repo '%s': %v", repo.Name, err)
				return
			}

			mux.Lock()
			refType := metadata.AttributeVcsRefHeadTypeBranch
			ados.mb.RecordVcsRefCountDataPoint(now, int64(len(branches)), repo.WebURL, repo.Name, repo.ID, refType)
			mux.Unlock()

			// Process branch metrics
			for _, branch := range branches {
				if branch.Name == repo.DefaultBranch {
					continue
				}

				// Calculate branch age
				if !branch.CreatedDate.IsZero() {
					branchAge := time.Since(branch.CreatedDate).Seconds()
					mux.Lock()
					ados.mb.RecordVcsRefTimeDataPoint(now, int64(branchAge), repo.WebURL, repo.Name, repo.ID, branch.Name, refType)
					mux.Unlock()
				}
			}

			// Get pull requests for this repository
			pullRequests, err := ados.getPullRequests(ctx, ados.cfg.Project, repo.ID)
			if err != nil {
				ados.logger.Sugar().Errorf("error getting pull requests for repo '%s': %v", repo.Name, err)
				return
			}

			// Process pull request metrics
			for _, pr := range pullRequests {
				mux.Lock()
				if pr.Status == "completed" {
					if !pr.ClosedDate.IsZero() && !pr.CreationDate.IsZero() {
						timeToMerge := int64(pr.ClosedDate.Sub(pr.CreationDate).Seconds())
						ados.mb.RecordVcsChangeTimeToMergeDataPoint(now, timeToMerge, repo.WebURL, repo.Name, repo.ID, pr.SourceRefName)
					}
				} else if pr.Status == "active" {
					if !pr.CreationDate.IsZero() {
						prAge := int64(time.Since(pr.CreationDate).Seconds())
						ados.mb.RecordVcsChangeDurationDataPoint(now, prAge, repo.WebURL, repo.Name, repo.ID,
							pr.SourceRefName, metadata.AttributeVcsChangeStateOpen)
					}
				}
				mux.Unlock()
			}
		}()
	}

	wg.Wait()

	ados.logger.Sugar().Infof("Finished processing Azure DevOps project %s in org %s", ados.cfg.Project, ados.cfg.Organization)

	// Set resource attributes
	ados.rb.SetVcsVendorName("azuredevops")
	ados.rb.SetOrganizationName(ados.cfg.Organization)

	res := ados.rb.Emit()
	return ados.mb.Emit(metadata.WithResource(res)), nil
}

// makeRequest makes an authenticated request to the Azure DevOps REST API
func (ados *azuredevopsScraper) makeRequest(ctx context.Context, endpoint string, baseUrlModifier string) (*http.Response, error) {
	baseModifier := ""
	if baseUrlModifier != "" {
		baseModifier = fmt.Sprintf("/%s/", baseUrlModifier)
	}
	fullURL := fmt.Sprintf("%s/%s%s/_apis/%s", ados.cfg.BaseURL, ados.cfg.Organization, baseModifier, endpoint)

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, err
	}

	// Add authentication header
	req.SetBasicAuth("", ados.cfg.PersonalAccessToken)
	req.Header.Set("Accept", "application/json")

	return ados.client.Do(req)
}

// getRepositories retrieves all repositories for a given project
func (ados *azuredevopsScraper) getRepositories(ctx context.Context, projectID string) ([]AzureDevOpsRepository, error) {
	resp, err := ados.makeRequest(ctx, "git/repositories?api-version=7.1", projectID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var result struct {
		Value []AzureDevOpsRepository `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Value, nil
}

// getBranches retrieves all branches for a given repository
func (ados *azuredevopsScraper) getBranches(ctx context.Context, projectID, repoID string) ([]AzureDevOpsBranch, error) {
	endpoint := fmt.Sprintf("git/repositories/%s/refs?api-version=7.1&filter=heads/", url.QueryEscape(repoID))
	resp, err := ados.makeRequest(ctx, endpoint, projectID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var result struct {
		Value []struct {
			Name     string `json:"name"`
			ObjectID string `json:"objectId"`
		} `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var branches []AzureDevOpsBranch
	for _, ref := range result.Value {
		// Extract branch name from refs/heads/branch-name
		branchName := ref.Name
		if len(branchName) > 11 && branchName[:11] == "refs/heads/" {
			branchName = branchName[11:]
		}

		branches = append(branches, AzureDevOpsBranch{
			Name:        branchName,
			ObjectID:    ref.ObjectID,
			CreatedDate: time.Now(), // Azure DevOps doesn't provide branch creation date directly
		})
	}

	return branches, nil
}

// getPullRequests retrieves pull requests for a given repository
func (ados *azuredevopsScraper) getPullRequests(ctx context.Context, projectID, repoID string) ([]AzureDevOpsPullRequest, error) {
	limit := ados.cfg.LimitPullRequests
	if limit <= 0 {
		limit = 100
	}

	endpoint := fmt.Sprintf("git/repositories/%s/pullrequests?api-version=7.1&$top=%d",
		url.QueryEscape(repoID), limit)

	resp, err := ados.makeRequest(ctx, endpoint, projectID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var result struct {
		Value []AzureDevOpsPullRequest `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Value, nil
}
