// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsscraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

// Release represents a release execution
type Release struct {
	ID        int          `json:"id"`
	Name      string       `json:"name"`
	CreatedOn NullableTime `json:"createdOn"`
}

// ReleaseEnvironmentDetail represents detailed environment info with deploy steps
type ReleaseEnvironmentDetail struct {
	ID          int          `json:"id"`
	ReleaseID   int          `json:"releaseId"`
	Name        string       `json:"name"`
	Status      string       `json:"status"`
	DeploySteps []DeployStep `json:"deploySteps"`
}

// DeployStep represents a deployment attempt
type DeployStep struct {
	ID                  int           `json:"id"`
	DeploymentID        int           `json:"deploymentId"`
	Attempt             int           `json:"attempt"`
	Reason              string        `json:"reason"`
	Status              string        `json:"status"`
	ReleaseDeployPhases []DeployPhase `json:"releaseDeployPhases"`
	QueuedOn            NullableTime  `json:"queuedOn"`
	StartedOn           NullableTime  `json:"startedOn"`
}

// DeployPhase represents a phase/stage within a deployment
type DeployPhase struct {
	ID             int             `json:"id"`
	PhaseID        string          `json:"phaseId"`
	Name           string          `json:"name"`
	Rank           int             `json:"rank"`
	PhaseType      string          `json:"phaseType"`
	Status         string          `json:"status"`
	StartedOn      NullableTime    `json:"startedOn"`
	DeploymentJobs []DeploymentJob `json:"deploymentJobs"`
}

// DeploymentJob represents a job within a deployment phase
type DeploymentJob struct {
	Job   ReleaseTask   `json:"job"`
	Tasks []ReleaseTask `json:"tasks"`
}

// ReleaseTask represents a task or job in a release deployment
type ReleaseTask struct {
	ID               int          `json:"id"`
	TimelineRecordID string       `json:"timelineRecordId"`
	Name             string       `json:"name"`
	DateStarted      NullableTime `json:"dateStarted"`
	DateEnded        NullableTime `json:"dateEnded"`
	StartTime        NullableTime `json:"startTime"`
	FinishTime       NullableTime `json:"finishTime"`
	Status           string       `json:"status"`
	PercentComplete  int          `json:"percentComplete"`
	Rank             int          `json:"rank"`
	AgentName        string       `json:"agentName"`
	LogURL           string       `json:"logUrl"`
}

// scrapeReleasePipelineMetrics scrapes release pipeline deployments and emits task-level logs
func (ados *azuredevopsScraper) scrapeReleasePipelineMetrics(ctx context.Context, logs plog.Logs) error {
	ados.logger.Sugar().Info("Scraping release pipeline metrics - STARTING")

	// Calculate lookback time window
	lookbackDays := ados.cfg.PipelineLookbackDays
	if lookbackDays <= 0 {
		lookbackDays = 30 // Default to 30 days
	}

	minTime := time.Now().AddDate(0, 0, -lookbackDays)
	ados.logger.Sugar().Debugf("Fetching release deployments from the last %d days (since %s)", lookbackDays, minTime.Format(time.RFC3339))

	// Fetch all release definitions
	releaseDefinitions, err := ados.fetchReleaseDefinitions(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch release definitions: %w", err)
	}

	ados.logger.Sugar().Infof("Found %d release definitions", len(releaseDefinitions))

	jobCount := 0
	taskCount := 0

	// For each release definition, fetch deployments
	for _, releaseDef := range releaseDefinitions {
		// Fetch releases for this definition
		releases, err := ados.fetchReleases(ctx, releaseDef.ID, minTime)
		if err != nil {
			ados.logger.Sugar().Errorf("Failed to fetch releases for definition %s: %v", releaseDef.Name, err)
			continue
		}

		ados.logger.Sugar().Debugf("Found %d releases for definition %s", len(releases), releaseDef.Name)

		// For each release, fetch environment details
		for _, release := range releases {
			environments, err := ados.fetchReleaseEnvironments(ctx, release.ID)
			if err != nil {
				ados.logger.Sugar().Errorf("Failed to fetch environments for release %s: %v", release.Name, err)
				continue
			}

			// Process each environment's deploy steps
			for _, env := range environments {
				for _, deployStep := range env.DeploySteps {
					// Process each phase in the deploy step
					for _, phase := range deployStep.ReleaseDeployPhases {
						// Process each deployment job
						for _, deployJob := range phase.DeploymentJobs {
							// Create job-level log record
							jobLogRecord := ados.createReleaseJobLogRecord(
								releaseDef,
								release,
								env,
								deployStep,
								phase,
								&deployJob,
							)
							jobLogRecord.CopyTo(logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty())
							jobCount++

							// Create log records for each task
							for _, task := range deployJob.Tasks {
								logRecord := ados.createReleaseTaskLogRecord(
									releaseDef,
									release,
									env,
									deployStep,
									phase,
									&task,
								)
								logRecord.CopyTo(logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty())
								taskCount++
							}
						}
					}
				}
			}
		}
	}

	ados.logger.Sugar().Infof("Scraped %d release pipeline job log records", jobCount)
	ados.logger.Sugar().Infof("Scraped %d release pipeline task log records", taskCount)
	return nil
}

// fetchReleaseDefinitions fetches all release definitions
func (ados *azuredevopsScraper) fetchReleaseDefinitions(ctx context.Context) ([]ReleaseDefinition, error) {
	apiURL := fmt.Sprintf("%s/%s/%s/_apis/release/definitions",
		vsrmBaseURL,
		url.PathEscape(ados.cfg.Organization),
		url.PathEscape(ados.cfg.Project))

	params := url.Values{}
	params.Add("api-version", apiVersion)
	if ados.cfg.ReleaseNameCriteria != nil {
		params.Add("searchText", ados.cfg.ReleaseNameCriteria)
	}

	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := ados.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release definitions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result ReleaseDefinitionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Value, nil
}

// fetchReleases fetches releases for a specific definition
func (ados *azuredevopsScraper) fetchReleases(ctx context.Context, definitionID int, minTime time.Time) ([]Release, error) {
	var allReleases []Release
	top := 50
	continuationToken := ""
	maxRuns := ados.cfg.MaxPipelineRuns

	for {
		apiURL := fmt.Sprintf("%s/%s/%s/_apis/release/releases",
			vsrmBaseURL,
			url.PathEscape(ados.cfg.Organization),
			url.PathEscape(ados.cfg.Project))

		params := url.Values{}
		params.Add("api-version", apiVersion)
		params.Add("definitionId", fmt.Sprintf("%d", definitionID))
		params.Add("$top", fmt.Sprintf("%d", top))
		if continuationToken != "" {
			params.Add("continuationToken", continuationToken)
		}

		fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := ados.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch releases: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}

		// Check for continuation token in response headers
		continuationToken = resp.Header.Get("x-ms-continuationtoken")

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		var result struct {
			Count int       `json:"count"`
			Value []Release `json:"value"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		pageSize := len(result.Value)

		// Filter releases by time and check if we should stop
		filteredCount := 0
		shouldStop := false
		for _, release := range result.Value {
			if !release.CreatedOn.Time.IsZero() && release.CreatedOn.Time.Before(minTime) {
				shouldStop = true
				break
			}
			allReleases = append(allReleases, release)
			filteredCount++
		}

		// Stop if no results, no continuation token, or found releases outside time window
		if pageSize == 0 || continuationToken == "" || shouldStop {
			break
		}

		if maxRuns > 0 && len(allReleases) >= maxRuns {
			allReleases = allReleases[:maxRuns]
			break
		}
	}

	return allReleases, nil
}

// fetchReleaseEnvironments fetches environment details for a release
func (ados *azuredevopsScraper) fetchReleaseEnvironments(ctx context.Context, releaseID int) ([]ReleaseEnvironmentDetail, error) {
	apiURL := fmt.Sprintf("%s/%s/%s/_apis/release/releases/%d",
		vsrmBaseURL,
		url.PathEscape(ados.cfg.Organization),
		url.PathEscape(ados.cfg.Project),
		releaseID)

	params := url.Values{}
	params.Add("api-version", apiVersion)
	params.Add("$expand", "environments")

	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := ados.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Environments []ReleaseEnvironmentDetail `json:"environments"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Environments, nil
}

// createReleaseJobLogRecord creates a log record for a release job
func (ados *azuredevopsScraper) createReleaseJobLogRecord(
	releaseDef ReleaseDefinition,
	release Release,
	env ReleaseEnvironmentDetail,
	deployStep DeployStep,
	phase DeployPhase,
	deployJob *DeploymentJob,
) plog.LogRecord {
	logRecord := plog.NewLogRecord()

	// Set timestamp to job finish time
	if !deployJob.Job.FinishTime.Time.IsZero() {
		logRecord.SetTimestamp(pcommon.NewTimestampFromTime(deployJob.Job.FinishTime.Time))
	} else if !deployJob.Job.DateEnded.Time.IsZero() {
		logRecord.SetTimestamp(pcommon.NewTimestampFromTime(deployJob.Job.DateEnded.Time))
	} else {
		logRecord.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	}

	logRecord.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	// Set severity based on job status
	severity := mapReleaseTaskStatusToSeverity(deployJob.Job.Status)
	logRecord.SetSeverityNumber(severity)
	logRecord.SetSeverityText(deployJob.Job.Status)

	// Set body
	logRecord.Body().SetStr(fmt.Sprintf("Release job '%s' %s", deployJob.Job.Name, deployJob.Job.Status))

	// Set attributes
	attrs := logRecord.Attributes()
	releaseJobToLogAttributes(releaseDef, release, env, deployStep, phase, deployJob, attrs)

	logRecord.SetDroppedAttributesCount(0)

	return logRecord
}

// createReleaseTaskLogRecord creates a log record for a release task
func (ados *azuredevopsScraper) createReleaseTaskLogRecord(
	releaseDef ReleaseDefinition,
	release Release,
	env ReleaseEnvironmentDetail,
	deployStep DeployStep,
	phase DeployPhase,
	task *ReleaseTask,
) plog.LogRecord {
	logRecord := plog.NewLogRecord()

	// Set timestamp to task finish time
	if !task.FinishTime.Time.IsZero() {
		logRecord.SetTimestamp(pcommon.NewTimestampFromTime(task.FinishTime.Time))
	} else if !task.DateEnded.Time.IsZero() {
		logRecord.SetTimestamp(pcommon.NewTimestampFromTime(task.DateEnded.Time))
	} else {
		logRecord.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	}

	logRecord.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	// Set severity based on task status
	severity := mapReleaseTaskStatusToSeverity(task.Status)
	logRecord.SetSeverityNumber(severity)
	logRecord.SetSeverityText(task.Status)

	// Set body
	logRecord.Body().SetStr(fmt.Sprintf("Release task '%s' %s", task.Name, task.Status))

	// Set attributes
	attrs := logRecord.Attributes()
	releaseTaskToLogAttributes(releaseDef, release, env, deployStep, phase, task, attrs)

	// Set resource attributes
	resourceAttrs := pcommon.NewMap()
	resourceAttrs.PutStr("vcs.provider.name", "azuredevops")
	resourceAttrs.PutStr("vcs.owner.name", ados.cfg.Organization)
	resourceAttrs.PutStr("azuredevops.project.name", ados.cfg.Project)

	// Set instrumentation scope
	logRecord.SetDroppedAttributesCount(0)

	return logRecord
}

// releaseTaskToLogAttributes maps release task fields to log attributes
func releaseTaskToLogAttributes(
	releaseDef ReleaseDefinition,
	release Release,
	env ReleaseEnvironmentDetail,
	deployStep DeployStep,
	phase DeployPhase,
	task *ReleaseTask,
	attrs pcommon.Map,
) {
	// Data stream attributes
	dataStream := attrs.PutEmptyMap("data_stream")
	dataStream.PutStr("type", "record")
	dataStream.PutStr("dataset", "pipeline-metrics")
	dataStream.PutStr("namespace", "azuredevops")

	// App/Release Definition info
	attrs.PutStr("app.name", releaseDef.Name)
	attrs.PutInt("release.definition.id", int64(releaseDef.ID))

	// Release info
	attrs.PutStr("release.name", release.Name)
	attrs.PutInt("release.id", int64(release.ID))

	// Environment info
	attrs.PutStr("environment.name", env.Name)
	attrs.PutInt("environment.id", int64(env.ID))
	attrs.PutStr("environment.status", env.Status)

	// Deployment info
	attrs.PutInt("deployment.id", int64(deployStep.DeploymentID))
	attrs.PutInt("deployment.attempt", int64(deployStep.Attempt))
	attrs.PutStr("deployment.reason", deployStep.Reason)
	attrs.PutStr("deployment.status", deployStep.Status)

	// Phase/Stage info
	attrs.PutStr("phase.name", phase.Name)
	attrs.PutStr("phase.id", phase.PhaseID)
	attrs.PutStr("phase.type", phase.PhaseType)
	attrs.PutStr("phase.status", phase.Status)
	attrs.PutInt("phase.rank", int64(phase.Rank))

	// Task info
	attrs.PutStr("record.type", "task")
	attrs.PutInt("task.id", int64(task.ID))
	attrs.PutStr("task.name", task.Name)
	attrs.PutStr("task.timeline_record_id", task.TimelineRecordID)
	attrs.PutStr("task.status", task.Status)
	attrs.PutInt("task.rank", int64(task.Rank))
	attrs.PutInt("task.percent_complete", int64(task.PercentComplete))

	if task.AgentName != "" {
		attrs.PutStr("agent.name", task.AgentName)
	}

	// Timing info
	if !task.StartTime.Time.IsZero() {
		attrs.PutStr("task.start_time", task.StartTime.Time.Format(time.RFC3339))
	} else if !task.DateStarted.Time.IsZero() {
		attrs.PutStr("task.start_time", task.DateStarted.Time.Format(time.RFC3339))
	}

	if !task.FinishTime.Time.IsZero() {
		attrs.PutStr("task.end_time", task.FinishTime.Time.Format(time.RFC3339))
	} else if !task.DateEnded.Time.IsZero() {
		attrs.PutStr("task.end_time", task.DateEnded.Time.Format(time.RFC3339))
	}

	// Calculate duration
	var startTime, endTime time.Time
	if !task.StartTime.Time.IsZero() {
		startTime = task.StartTime.Time
	} else if !task.DateStarted.Time.IsZero() {
		startTime = task.DateStarted.Time
	}

	if !task.FinishTime.Time.IsZero() {
		endTime = task.FinishTime.Time
	} else if !task.DateEnded.Time.IsZero() {
		endTime = task.DateEnded.Time
	}

	if !startTime.IsZero() && !endTime.IsZero() {
		duration := endTime.Sub(startTime).Seconds()
		attrs.PutDouble("task.duration_seconds", duration)
	}

	// Calculate queue time (from deploy step queued to phase started)
	if !deployStep.QueuedOn.Time.IsZero() && !phase.StartedOn.Time.IsZero() {
		queueTime := phase.StartedOn.Time.Sub(deployStep.QueuedOn.Time).Seconds()
		attrs.PutDouble("deployment.queue_time_seconds", queueTime)
	}

	// Status attribute for compatibility
	attrs.PutStr("status", task.Status)
}

// releaseJobToLogAttributes maps release job fields to log attributes
func releaseJobToLogAttributes(
	releaseDef ReleaseDefinition,
	release Release,
	env ReleaseEnvironmentDetail,
	deployStep DeployStep,
	phase DeployPhase,
	deployJob *DeploymentJob,
	attrs pcommon.Map,
) {
	// Data stream attributes
	dataStream := attrs.PutEmptyMap("data_stream")
	dataStream.PutStr("type", "record")
	dataStream.PutStr("dataset", "pipeline-metrics")
	dataStream.PutStr("namespace", "azuredevops")

	// App/Release Definition info
	attrs.PutStr("app.name", releaseDef.Name)
	attrs.PutInt("release.definition.id", int64(releaseDef.ID))

	// Release info
	attrs.PutStr("release.name", release.Name)
	attrs.PutInt("release.id", int64(release.ID))

	// Environment info
	attrs.PutStr("environment.name", env.Name)
	attrs.PutInt("environment.id", int64(env.ID))
	attrs.PutStr("environment.status", env.Status)

	// Deployment info
	attrs.PutInt("deployment.id", int64(deployStep.DeploymentID))
	attrs.PutInt("deployment.attempt", int64(deployStep.Attempt))
	attrs.PutStr("deployment.reason", deployStep.Reason)
	attrs.PutStr("deployment.status", deployStep.Status)

	// Phase/Stage info
	attrs.PutStr("phase.name", phase.Name)
	attrs.PutStr("phase.id", phase.PhaseID)
	attrs.PutStr("phase.type", phase.PhaseType)
	attrs.PutStr("phase.status", phase.Status)
	attrs.PutInt("phase.rank", int64(phase.Rank))

	// Job info
	attrs.PutStr("record.type", "job")
	attrs.PutInt("job.id", int64(deployJob.Job.ID))
	attrs.PutStr("job.name", deployJob.Job.Name)
	attrs.PutStr("job.timeline_record_id", deployJob.Job.TimelineRecordID)
	attrs.PutStr("job.status", deployJob.Job.Status)
	attrs.PutInt("job.rank", int64(deployJob.Job.Rank))

	if deployJob.Job.AgentName != "" {
		attrs.PutStr("agent.name", deployJob.Job.AgentName)
	}

	// Timing info
	var startTime, endTime time.Time
	if !deployJob.Job.StartTime.Time.IsZero() {
		startTime = deployJob.Job.StartTime.Time
		attrs.PutStr("job.start_time", startTime.Format(time.RFC3339))
	} else if !deployJob.Job.DateStarted.Time.IsZero() {
		startTime = deployJob.Job.DateStarted.Time
		attrs.PutStr("job.start_time", startTime.Format(time.RFC3339))
	}

	if !deployJob.Job.FinishTime.Time.IsZero() {
		endTime = deployJob.Job.FinishTime.Time
		attrs.PutStr("job.end_time", endTime.Format(time.RFC3339))
	} else if !deployJob.Job.DateEnded.Time.IsZero() {
		endTime = deployJob.Job.DateEnded.Time
		attrs.PutStr("job.end_time", endTime.Format(time.RFC3339))
	}

	// Calculate job duration
	if !startTime.IsZero() && !endTime.IsZero() {
		duration := endTime.Sub(startTime).Seconds()
		attrs.PutDouble("job.duration_seconds", duration)
	}

	// Calculate queue time (from deploy step queued to job started)
	if !deployStep.QueuedOn.Time.IsZero() && !startTime.IsZero() {
		queueTime := startTime.Sub(deployStep.QueuedOn.Time).Seconds()
		attrs.PutDouble("job.queue_time_seconds", queueTime)
	}

	// Status attribute for compatibility
	attrs.PutStr("status", deployJob.Job.Status)
}

// mapReleaseTaskStatusToSeverity maps release task status to OTel severity
func mapReleaseTaskStatusToSeverity(status string) plog.SeverityNumber {
	switch status {
	case "succeeded":
		return plog.SeverityNumberInfo
	case "failed":
		return plog.SeverityNumberError
	case "canceled", "cancelled":
		return plog.SeverityNumberWarn
	case "skipped":
		return plog.SeverityNumberDebug
	case "inProgress", "pending":
		return plog.SeverityNumberInfo2
	default:
		return plog.SeverityNumberUnspecified
	}
}
