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

// deploymentMetricKey represents a unique key for deployment count and timestamp metrics
type deploymentMetricKey struct {
	Service     string
	Environment string
	Status      string
}

// deploymentDurationKey represents a unique key for deployment duration metrics
type deploymentDurationKey struct {
	Service     string
	Environment string
}

// WorkItem represents an Azure DevOps work item
type WorkItem struct {
	ID     int                    `json:"id"`
	Fields map[string]interface{} `json:"fields"`
}

// WorkItemQueryResult represents the result of a WIQL query
type WorkItemQueryResult struct {
	WorkItems []struct {
		ID  int    `json:"id"`
		URL string `json:"url"`
	} `json:"workItems"`
}

// WorkItemBatchResult represents a batch get work items response
type WorkItemBatchResult struct {
	Count int        `json:"count"`
	Value []WorkItem `json:"value"`
}

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
	extensions := host.GetExtensions()
	ados.client, err = ados.cfg.ToClient(ctx, extensions, ados.settings)
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
		deployments, err := ados.fetchDeploymentData(ctx)
		if err != nil {
			ados.logger.Sugar().Errorf("error fetching deployments: %v", err)
		} else {
			ados.recordDeploymentMetrics(now, deployments, ados.cfg.DeploymentStageName)
		}
	}

	// Scrape work item metrics if enabled
	if ados.cfg.WorkItemsEnabled {
		workItems, err := ados.fetchWorkItems(ctx, ados.cfg.Organization, ados.cfg.Project, ados.cfg.WorkItemLookbackDays)
		if err != nil {
			ados.logger.Sugar().Errorf("error fetching work items: %v", err)
		} else {
			ados.recordWorkItemMetrics(now, workItems, ados.cfg.Project)
		}
	}

	// Set resource attributes
	ados.rb.SetVcsProviderName("azuredevops")
	ados.rb.SetVcsOwnerName(ados.cfg.Organization)

	res := ados.rb.Emit()
	return ados.mb.Emit(metadata.WithResource(res)), nil
}

// fetchDeploymentData fetches deployment data from Azure DevOps Release Management API
func (ados *azuredevopsScraper) fetchDeploymentData(ctx context.Context) ([]Deployment, error) {
	// Always fetch deployments from the configured lookback window
	// This ensures metrics are consistently updated even when no new deployments occur
	lookbackDays := ados.cfg.DeploymentLookbackDays
	if lookbackDays <= 0 {
		lookbackDays = 30 // default to 30 days
	}
	minTime := time.Now().UTC().AddDate(0, 0, -lookbackDays)
	ados.logger.Sugar().Infof("Fetching deployments from last %d days", lookbackDays)

	ados.logger.Sugar().Infof("Fetching deployments for pipeline '%s', stage '%s'", ados.cfg.DeploymentPipelineName, ados.cfg.DeploymentStageName)

	// Get release definition ID
	definitionID, err := ados.getReleaseDefinitionID(ctx, ados.cfg.Organization, ados.cfg.Project, ados.cfg.DeploymentPipelineName)
	if err != nil {
		return nil, fmt.Errorf("failed to get release definition ID: %w", err)
	}
	ados.logger.Sugar().Debugf("Found release definition ID: %d", definitionID)

	// Get environment ID
	environmentID, err := ados.getDefinitionEnvironmentID(ctx, ados.cfg.Organization, ados.cfg.Project, definitionID, ados.cfg.DeploymentStageName)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment ID: %w", err)
	}
	ados.logger.Sugar().Debugf("Found environment ID: %d", environmentID)

	// Fetch deployments since minTime
	deployments, err := ados.fetchDeployments(ctx, ados.cfg.Organization, ados.cfg.Project, definitionID, environmentID, minTime)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch deployments: %w", err)
	}
	ados.logger.Sugar().Infof("Fetched %d deployments", len(deployments))

	return deployments, nil
}

// recordDeploymentMetrics processes deployments and records metrics
func (ados *azuredevopsScraper) recordDeploymentMetrics(now pcommon.Timestamp, deployments []Deployment, environment string) {
	// Track counts, durations, and last timestamps using typed keys
	counts := make(map[deploymentMetricKey]int64)
	durations := make(map[deploymentDurationKey][]int64)
	lastTimestamps := make(map[deploymentMetricKey]int64)

	for _, deployment := range deployments {
		service := extractServiceName(deployment)
		rawStatus := deployment.DeploymentStatus

		// Only process completed deployments with final outcomes
		// Skip in-progress/undefined as they don't represent final outcomes
		// Include notDeployed as it represents a deployment failure (queued but never executed)
		normalizedStatus := strings.ToLower(strings.TrimSpace(rawStatus))

		// Map partiallySucceeded and any other non-success status to "failed"
		var status string
		if normalizedStatus == "succeeded" {
			status = "succeeded"
		} else if normalizedStatus == "partiallysucceeded" || normalizedStatus == "failed" || normalizedStatus == "notdeployed" {
			status = "failed"
		} else {
			ados.logger.Sugar().Debugf("Skipping deployment ID %d with non-final status: %q (service: %s)", deployment.ID, rawStatus, service)
			continue
		}

		// Count deployments
		countKey := deploymentMetricKey{
			Service:     service,
			Environment: environment,
			Status:      status,
		}
		counts[countKey]++

		// Track duration for succeeded deployments
		if status == "succeeded" && !deployment.StartedOn.IsZero() && !deployment.CompletedOn.IsZero() {
			duration := int64(deployment.CompletedOn.Time.Sub(deployment.StartedOn.Time).Seconds())
			durationKey := deploymentDurationKey{
				Service:     service,
				Environment: environment,
			}
			durations[durationKey] = append(durations[durationKey], duration)
		}

		// Track last deployment timestamp
		if !deployment.CompletedOn.IsZero() {
			timestampKey := deploymentMetricKey{
				Service:     service,
				Environment: environment,
				Status:      status,
			}
			timestamp := deployment.CompletedOn.Unix()
			if existing, ok := lastTimestamps[timestampKey]; !ok || timestamp > existing {
				lastTimestamps[timestampKey] = timestamp
			}
		}
	}

	// Record deployment counts
	for key, count := range counts {
		statusAttr := mapDeploymentStatus(key.Status)
		ados.mb.RecordDeployDeploymentCountDataPoint(now, count, key.Service, key.Environment, statusAttr)
	}

	// Record average durations
	for key, durationList := range durations {
		if len(durationList) > 0 {
			var sum int64
			for _, d := range durationList {
				sum += d
			}
			avgDuration := sum / int64(len(durationList))
			ados.mb.RecordDeployDeploymentAverageDurationDataPoint(now, avgDuration, key.Service, key.Environment)
		}
	}

	// Record last timestamps
	for key, timestamp := range lastTimestamps {
		statusAttr := mapDeploymentStatus(key.Status)
		ados.mb.RecordDeployDeploymentLastTimestampDataPoint(now, timestamp, key.Service, key.Environment, statusAttr)
	}

	ados.logger.Sugar().Infof("Recorded deployment metrics: %d count entries, %d duration entries, %d timestamp entries",
		len(counts), len(durations), len(lastTimestamps))
}

// mapDeploymentStatus converts Azure DevOps deployment status string to OTel semantic convention enum
// Only maps completed deployments (succeeded/failed) per OTel spec
func mapDeploymentStatus(status string) metadata.AttributeDeploymentStatus {
	// Normalize status string
	normalized := strings.ToLower(strings.TrimSpace(status))

	// Map to OTel semantic convention values
	switch normalized {
	case "succeeded":
		return metadata.AttributeDeploymentStatusSucceeded
	case "failed":
		return metadata.AttributeDeploymentStatusFailed
	default:
		// Should not reach here due to filtering in recordDeploymentMetrics
		return metadata.AttributeDeploymentStatusFailed
	}
}
