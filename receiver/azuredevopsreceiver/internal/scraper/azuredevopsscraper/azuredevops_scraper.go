// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsscraper

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
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

	// Scrape deployment metrics if configured
	if ados.cfg.DeploymentPipelineName != "" && ados.cfg.DeploymentStageName != "" {
		if err := ados.scrapeDeployments(ctx, now); err != nil {
			ados.logger.Sugar().Errorf("error scraping deployments: %v", err)
		}
	}

	// Set resource attributes
	ados.rb.SetVcsProviderName("azuredevops")
	ados.rb.SetVcsOwnerName(ados.cfg.Organization)

	res := ados.rb.Emit()
	return ados.mb.Emit(metadata.WithResource(res)), nil
}

// scrapeDeployments fetches and records deployment metrics
func (ados *azuredevopsScraper) scrapeDeployments(ctx context.Context, now pcommon.Timestamp) error {
	lookbackDays := ados.cfg.DeploymentLookbackDays
	if lookbackDays <= 0 {
		lookbackDays = 30 // default to 30 days
	}

	ados.logger.Sugar().Infof("Scraping deployments for pipeline '%s', stage '%s'", ados.cfg.DeploymentPipelineName, ados.cfg.DeploymentStageName)

	// Get release definition ID
	definitionID, err := ados.getReleaseDefinitionID(ctx, ados.cfg.Organization, ados.cfg.Project, ados.cfg.DeploymentPipelineName)
	if err != nil {
		return fmt.Errorf("failed to get release definition ID: %w", err)
	}
	ados.logger.Sugar().Debugf("Found release definition ID: %d", definitionID)

	// Get environment ID
	environmentID, err := ados.getDefinitionEnvironmentID(ctx, ados.cfg.Organization, ados.cfg.Project, definitionID, ados.cfg.DeploymentStageName)
	if err != nil {
		return fmt.Errorf("failed to get environment ID: %w", err)
	}
	ados.logger.Sugar().Debugf("Found environment ID: %d", environmentID)

	// Fetch deployments
	deployments, err := ados.fetchDeployments(ctx, ados.cfg.Organization, ados.cfg.Project, definitionID, environmentID, lookbackDays)
	if err != nil {
		return fmt.Errorf("failed to fetch deployments: %w", err)
	}
	ados.logger.Sugar().Infof("Fetched %d deployments", len(deployments))

	// Process and record metrics
	ados.recordDeploymentMetrics(now, deployments, ados.cfg.DeploymentStageName)

	return nil
}

// recordDeploymentMetrics processes deployments and records metrics
func (ados *azuredevopsScraper) recordDeploymentMetrics(now pcommon.Timestamp, deployments []Deployment, environment string) {
	// Track counts, durations, and last timestamps
	counts := make(map[string]int64)         // key: service|environment|status
	durations := make(map[string][]int64)    // key: service|environment
	lastTimestamps := make(map[string]int64) // key: service|environment|status

	for _, deployment := range deployments {
		service := extractServiceName(deployment)
		status := deployment.DeploymentStatus

		// Count deployments
		countKey := fmt.Sprintf("%s|%s|%s", service, environment, status)
		counts[countKey]++

		// Track duration for succeeded deployments
		if status == "succeeded" && !deployment.StartedOn.IsZero() && !deployment.CompletedOn.IsZero() {
			duration := int64(deployment.CompletedOn.Sub(deployment.StartedOn).Seconds())
			durationKey := fmt.Sprintf("%s|%s", service, environment)
			durations[durationKey] = append(durations[durationKey], duration)
		}

		// Track last deployment timestamp
		if !deployment.CompletedOn.IsZero() {
			timestampKey := fmt.Sprintf("%s|%s|%s", service, environment, status)
			timestamp := deployment.CompletedOn.Unix()
			if existing, ok := lastTimestamps[timestampKey]; !ok || timestamp > existing {
				lastTimestamps[timestampKey] = timestamp
			}
		}
	}

	// Record deployment counts
	for key, count := range counts {
		parts := strings.Split(key, "|")
		if len(parts) == 3 {
			statusAttr := mapDeploymentStatus(parts[2])
			ados.mb.RecordDeployDeploymentCountDataPoint(now, count, parts[0], parts[1], statusAttr)
		}
	}

	// Record average durations
	for key, durationList := range durations {
		if len(durationList) > 0 {
			var sum int64
			for _, d := range durationList {
				sum += d
			}
			avgDuration := sum / int64(len(durationList))
			parts := strings.Split(key, "|")
			if len(parts) == 2 {
				ados.mb.RecordDeployDeploymentDurationDataPoint(now, avgDuration, parts[0], parts[1])
			}
		}
	}

	// Record last timestamps
	for key, timestamp := range lastTimestamps {
		parts := strings.Split(key, "|")
		if len(parts) == 3 {
			statusAttr := mapDeploymentStatus(parts[2])
			ados.mb.RecordDeployDeploymentLastTimestampDataPoint(now, timestamp, parts[0], parts[1], statusAttr)
		}
	}

	ados.logger.Sugar().Infof("Recorded deployment metrics: %d count entries, %d duration entries, %d timestamp entries",
		len(counts), len(durations), len(lastTimestamps))
}

// mapDeploymentStatus converts Azure DevOps deployment status string to typed enum
func mapDeploymentStatus(status string) metadata.AttributeDeployStatus {
	// Normalize status string
	normalized := strings.ToLower(strings.TrimSpace(status))

	// Map to enum values defined in metadata
	switch normalized {
	case "succeeded":
		return metadata.AttributeDeployStatusSucceeded
	case "failed":
		return metadata.AttributeDeployStatusFailed
	case "inprogress", "in_progress":
		return metadata.AttributeDeployStatusInProgress
	default:
		// Default to inProgress for unknown statuses
		return metadata.AttributeDeployStatusInProgress
	}
}
