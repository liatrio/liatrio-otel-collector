// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsscraper

import (
	"context"
	"errors"
	"net/http"
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
		ados.logger.Sugar().Errorf("error getting repositories for project %s: %v", ados.cfg.Project, err)
		return ados.mb.Emit(), err
	}

	ados.logger.Sugar().Infof("Found %d repositories for organization %s and project %s", len(repos), ados.cfg.Organization, ados.cfg.Project)

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
			refType := metadata.AttributeVcsRefTypeBranch
			ados.mb.RecordVcsRefCountDataPoint(now, int64(len(branches)), repo.WebURL, repo.Name, repo.ID, refType)
			mux.Unlock()

			// Process branch metrics
			for _, branch := range branches {
				if branch.Name == repo.DefaultBranch {
					continue
				}

				// Calculate branch age by finding the initial commit that diverged from default branch
				commit, err := ados.getInitialCommit(ctx, ados.cfg.Project, repo.ID, repo.DefaultBranch, branch.Name)
				if err != nil {
					ados.logger.Sugar().Errorf("error getting initial commit for repo '%s' and branch '%s': %v", repo.Name, branch.Name, err)
				}

				if commit != nil {
					branchAge := time.Since(commit.Author.Date).Seconds()
					mux.Lock()
					ados.mb.RecordVcsRefTimeDataPoint(now, int64(branchAge), repo.WebURL, repo.Name, repo.ID, branch.Name, refType)
					mux.Unlock()
				}
			}

			latestBuildId, err := ados.getLatestBuildId(ctx, ados.cfg.Project, repo.ID, repo.DefaultBranch)
			if err != nil {
				ados.logger.Sugar().Errorf("error getting latest build id for repo '%s': %v", repo.Name, err)
			}

			if latestBuildId != "" {
				coveragePercentage, err := ados.getCodeCoverageForBuild(ctx, ados.cfg.Project, latestBuildId)
				if err != nil {
					ados.logger.Sugar().Errorf("error getting code coverage for build '%s': %v", latestBuildId, err)
				}
				ados.logger.Sugar().Infof("Recording '%d%%' code coverage for build '%s' in repo '%s'", coveragePercentage, latestBuildId, repo.Name)
				ados.mb.RecordVcsCodeCoverageDataPoint(
					now,
					coveragePercentage,
					repo.WebURL,
					repo.Name,
					repo.ID,
					repo.DefaultBranch,
					metadata.AttributeVcsRefHeadTypeBranch,
				)
			}

			// Get pull requests for this repository
			pullRequests, err := ados.getCombinedPullRequests(ctx, ados.cfg.Project, repo.ID)
			if err != nil {
				ados.logger.Sugar().Errorf("error getting pull requests for repo '%s': %v", repo.Name, err)
				return
			}

			// Count PRs by state
			openCount := int64(0)
			mergedCount := int64(0)

			// Process pull request metrics
			for _, pr := range pullRequests {
				mux.Lock()
				switch pr.Status {
				case "completed":
					mergedCount++
					if !pr.ClosedDate.IsZero() && !pr.CreationDate.IsZero() {
						timeToMerge := int64(pr.ClosedDate.Sub(pr.CreationDate).Seconds())
						ados.mb.RecordVcsChangeTimeToMergeDataPoint(now, timeToMerge, repo.WebURL, repo.Name, repo.ID, pr.SourceRefName)
					}
				case "active":
					openCount++
					if !pr.CreationDate.IsZero() {
						prAge := int64(time.Since(pr.CreationDate).Seconds())
						ados.mb.RecordVcsChangeDurationDataPoint(now, prAge, repo.WebURL, repo.Name, repo.ID,
							pr.SourceRefName, metadata.AttributeVcsChangeStateOpen)
					}
				}
				mux.Unlock()
			}

			// Record PR counts by state
			mux.Lock()
			if openCount > 0 {
				ados.mb.RecordVcsChangeCountDataPoint(now, openCount, repo.WebURL, metadata.AttributeVcsChangeStateOpen, repo.Name, repo.ID)
			}
			if mergedCount > 0 {
				ados.mb.RecordVcsChangeCountDataPoint(now, mergedCount, repo.WebURL, metadata.AttributeVcsChangeStateMerged, repo.Name, repo.ID)
			}
			mux.Unlock()
		}()
	}

	wg.Wait()

	ados.logger.Sugar().Infof("Finished processing Azure DevOps project %s in org %s", ados.cfg.Project, ados.cfg.Organization)

	// Set resource attributes
	ados.rb.SetVcsProviderName("azuredevops")
	ados.rb.SetVcsOwnerName(ados.cfg.Organization)

	res := ados.rb.Emit()
	return ados.mb.Emit(metadata.WithResource(res)), nil
}
