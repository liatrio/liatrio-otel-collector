// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuredevopsscraper

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

// scrapeLogs scrapes pipeline metrics and emits them as logs
func (ados *azuredevopsScraper) scrapeLogs(ctx context.Context) (plog.Logs, error) {
	ados.logger.Sugar().Debug("scrapeLogs called - pipeline metrics scraping not yet implemented")
	
	// Return empty logs for now - will be implemented in Task 4.0
	return plog.NewLogs(), nil
}

// createLogRecordFromJob creates an OpenTelemetry log record from a pipeline job
func createLogRecordFromJob(
	buildRun *BuildRun,
	job *BuildTimelineRecord,
	logs plog.Logs,
) {
	// Create resource logs
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	
	// Set resource attributes
	resourceAttrs := resourceLogs.Resource().Attributes()
	resourceAttrs.PutStr("vcs.provider.name", "azuredevops")
	resourceAttrs.PutStr("vcs.owner.name", buildRun.Project.Name)
	resourceAttrs.PutStr("azuredevops.project.id", buildRun.Project.ID)
	resourceAttrs.PutStr("azuredevops.project.name", buildRun.Project.Name)
	
	// Create scope logs
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	scopeLogs.Scope().SetName("azuredevops.pipeline.job")
	scopeLogs.Scope().SetVersion("1.0.0")
	
	// Create log record
	logRecord := scopeLogs.LogRecords().AppendEmpty()
	
	// Set timestamp
	if !job.FinishTime.IsZero() {
		logRecord.SetTimestamp(pcommon.NewTimestampFromTime(job.FinishTime.Time))
	} else if !job.StartTime.IsZero() {
		logRecord.SetTimestamp(pcommon.NewTimestampFromTime(job.StartTime.Time))
	} else {
		logRecord.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	}
	
	// Set observed timestamp
	logRecord.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	
	// Set severity based on job result
	logRecord.SetSeverityNumber(mapJobResultToSeverity(job.Result))
	logRecord.SetSeverityText(job.Result)
	
	// Set log body
	logRecord.Body().SetStr(fmt.Sprintf("Pipeline job '%s' %s", job.Name, job.Result))
	
	// Set all job attributes
	attrs := logRecord.Attributes()
	jobToLogAttributes(buildRun, job, attrs)
}

// jobToLogAttributes maps all job fields to log attributes
func jobToLogAttributes(buildRun *BuildRun, job *BuildTimelineRecord, attrs pcommon.Map) {
	// Job identification
	attrs.PutStr("job.id", job.ID)
	attrs.PutStr("job.name", job.Name)
	attrs.PutStr("job.identifier", job.Identifier)
	attrs.PutInt("job.attempt", int64(job.Attempt))
	
	// Pipeline information
	attrs.PutStr("pipeline.id", fmt.Sprintf("%d", buildRun.Definition.ID))
	attrs.PutStr("pipeline.name", buildRun.Definition.Name)
	attrs.PutStr("pipeline.run.id", fmt.Sprintf("%d", buildRun.ID))
	attrs.PutStr("pipeline.run.number", buildRun.BuildNumber)
	attrs.PutStr("pipeline.run.url", buildRun.URL)
	
	// Stage information (from parent)
	if job.ParentID != "" {
		attrs.PutStr("stage.id", job.ParentID)
		// Stage name would need to be looked up from timeline, using parent ID for now
		attrs.PutStr("stage.name", job.ParentID)
	}
	
	// Job status and result
	attrs.PutStr("job.state", job.State)
	attrs.PutStr("job.result", job.Result)
	attrs.PutStr("status", mapJobResult(job.Result))
	
	// Timing information
	if !job.StartTime.IsZero() {
		attrs.PutStr("job.start_time", job.StartTime.Format(time.RFC3339))
	}
	if !job.FinishTime.IsZero() {
		attrs.PutStr("job.end_time", job.FinishTime.Format(time.RFC3339))
	}
	
	// Calculate duration
	if !job.StartTime.IsZero() && !job.FinishTime.IsZero() {
		duration := job.FinishTime.Sub(job.StartTime.Time)
		attrs.PutDouble("job.duration_seconds", duration.Seconds())
	}
	
	// Queue time calculation
	if !buildRun.QueueTime.IsZero() && !buildRun.StartTime.IsZero() {
		queueDuration := buildRun.StartTime.Sub(buildRun.QueueTime.Time)
		attrs.PutDouble("job.queue_time_seconds", queueDuration.Seconds())
	}
	
	// Branch and repository information
	attrs.PutStr("vcs.ref.name", buildRun.SourceBranch)
	attrs.PutStr("vcs.ref.revision", buildRun.SourceVersion) // commit SHA
	attrs.PutStr("vcs.repository.name", buildRun.Repository.Name)
	attrs.PutStr("vcs.repository.id", buildRun.Repository.ID)
	attrs.PutStr("vcs.repository.url", buildRun.Repository.URL)
	
	// Trigger information
	attrs.PutStr("pipeline.triggered_by", buildRun.RequestedFor.DisplayName)
	attrs.PutStr("pipeline.triggered_by.email", buildRun.RequestedFor.UniqueName)
	attrs.PutStr("pipeline.trigger_reason", buildRun.Reason)
	
	// Agent/worker information
	if job.WorkerName != "" {
		attrs.PutStr("job.worker_name", job.WorkerName)
	}
	
	// Error and warning counts
	attrs.PutInt("job.error_count", int64(job.ErrorCount))
	attrs.PutInt("job.warning_count", int64(job.WarningCount))
	
	// Order in execution
	attrs.PutInt("job.order", int64(job.Order))
}

// mapJobResultToSeverity maps Azure DevOps job result to OpenTelemetry severity
func mapJobResultToSeverity(result string) plog.SeverityNumber {
	switch result {
	case "succeeded":
		return plog.SeverityNumberInfo
	case "succeededWithIssues":
		return plog.SeverityNumberWarn
	case "failed":
		return plog.SeverityNumberError
	case "canceled", "cancelled":
		return plog.SeverityNumberWarn
	case "skipped":
		return plog.SeverityNumberInfo
	case "abandoned":
		return plog.SeverityNumberWarn
	default:
		return plog.SeverityNumberUnspecified
	}
}

// mapJobResult normalizes job result to a standard status
func mapJobResult(result string) string {
	switch result {
	case "succeeded":
		return "succeeded"
	case "succeededWithIssues":
		return "succeeded_with_issues"
	case "failed":
		return "failed"
	case "canceled", "cancelled":
		return "canceled"
	case "skipped":
		return "skipped"
	case "abandoned":
		return "abandoned"
	default:
		return "unknown"
	}
}
