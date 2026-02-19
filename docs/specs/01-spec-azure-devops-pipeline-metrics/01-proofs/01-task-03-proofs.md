# Task 3.0 Proof Artifacts: Add OpenTelemetry Logs Support to Azure DevOps Receiver

**Date**: 2026-02-19  
**Task**: 3.0 - Add OpenTelemetry Logs Support to Azure DevOps Receiver  
**Status**: ✅ Complete

## Overview

Successfully extended the Azure DevOps receiver to support the OpenTelemetry logs signal. The implementation includes factory modifications, scraper interface extensions, and comprehensive log record creation infrastructure ready for pipeline metrics integration in Task 4.0.

## Proof Artifacts

### 1. Factory Logs Receiver Support

**File**: `receiver/azuredevopsreceiver/factory.go`

Added logs receiver capability to the factory:

```go
// NewFactory creates a factory for the azuredevops receiver
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, metadata.MetricsStability),
		receiver.WithTraces(createTracesReceiver, metadata.TracesStability),
		receiver.WithLogs(createLogsReceiver, metadata.LogsStability),  // ✅ Added
	)
}

// createLogsReceiver creates a logs receiver based on provided config.
func createLogsReceiver(
	ctx context.Context,
	params receiver.Settings,
	cfg component.Config,
	consumer consumer.Logs,
) (receiver.Logs, error) {
	// Implementation creates logs scraper and controller
	// Returns scraperhelper.NewLogsController
}
```

**Verification**: ✅ Build successful, factory supports logs signal

### 2. Metadata Logs Stability Constant

**File**: `receiver/azuredevopsreceiver/internal/metadata/generated_status.go`

```go
const (
	MetricsStability = component.StabilityLevelDevelopment
	TracesStability  = component.StabilityLevelDevelopment
	LogsStability    = component.StabilityLevelDevelopment  // ✅ Added
)
```

**Verification**: ✅ Logs stability level defined

### 3. ScraperFactory Interface Extension

**File**: `receiver/azuredevopsreceiver/internal/scraper.go`

Extended interface to support logs scraper creation:

```go
type ScraperFactory interface {
	CreateDefaultConfig() Config
	CreateMetricsScraper(ctx context.Context, params receiver.Settings, cfg Config) (scraper.Metrics, error)
	CreateLogsScraper(ctx context.Context, params receiver.Settings, cfg Config) (scraper.Logs, error)  // ✅ Added
}
```

**Verification**: ✅ Interface supports both metrics and logs scrapers

### 4. Logs Scraper Factory Implementation

**File**: `receiver/azuredevopsreceiver/internal/scraper/azuredevopsscraper/factory.go`

```go
func (f *Factory) CreateLogsScraper(
	ctx context.Context,
	params receiver.Settings,
	cfg internal.Config,
) (scraper.Logs, error) {
	conf := cfg.(*Config)
	s := newAzureDevOpsScraper(ctx, params, conf)

	return scraper.NewLogs(
		s.scrapeLogs,
		scraper.WithStart(s.start),
	)
}
```

**Verification**: ✅ Factory creates logs scraper using existing scraper infrastructure

### 5. Log Record Creation Infrastructure

**File**: `receiver/azuredevopsreceiver/internal/scraper/azuredevopsscraper/logs.go` (178 lines)

#### 5.1 Main Scrape Method (Stub for Task 4.0)

```go
func (ados *azuredevopsScraper) scrapeLogs(ctx context.Context) (plog.Logs, error) {
	ados.logger.Sugar().Debug("scrapeLogs called - pipeline metrics scraping not yet implemented")
	
	// Return empty logs for now - will be implemented in Task 4.0
	return plog.NewLogs(), nil
}
```

**Purpose**: Entry point for logs scraping, ready for pipeline integration

#### 5.2 Log Record Builder

```go
func createLogRecordFromJob(
	buildRun *BuildRun,
	job *BuildTimelineRecord,
	logs plog.Logs,
) {
	// Creates resource logs with resource attributes
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	resourceAttrs := resourceLogs.Resource().Attributes()
	resourceAttrs.PutStr("vcs.provider.name", "azuredevops")
	resourceAttrs.PutStr("vcs.owner.name", buildRun.Project.Name)
	resourceAttrs.PutStr("azuredevops.project.id", buildRun.Project.ID)
	resourceAttrs.PutStr("azuredevops.project.name", buildRun.Project.Name)
	
	// Creates scope logs
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	scopeLogs.Scope().SetName("azuredevops.pipeline.job")
	scopeLogs.Scope().SetVersion("1.0.0")
	
	// Creates log record with timestamps, severity, body, and attributes
	logRecord := scopeLogs.LogRecords().AppendEmpty()
	// ... (full implementation in file)
}
```

**Features**:
- ✅ Resource attributes for VCS provider and project
- ✅ Scope identification for pipeline jobs
- ✅ Timestamp handling (finish time, start time, or current)
- ✅ Observed timestamp
- ✅ Severity mapping based on job result
- ✅ Descriptive log body
- ✅ Comprehensive attribute mapping

#### 5.3 Attribute Mapping (25+ Fields)

```go
func jobToLogAttributes(buildRun *BuildRun, job *BuildTimelineRecord, attrs pcommon.Map) {
	// Job identification (4 fields)
	attrs.PutStr("job.id", job.ID)
	attrs.PutStr("job.name", job.Name)
	attrs.PutStr("job.identifier", job.Identifier)
	attrs.PutInt("job.attempt", int64(job.Attempt))
	
	// Pipeline information (5 fields)
	attrs.PutStr("pipeline.id", fmt.Sprintf("%d", buildRun.Definition.ID))
	attrs.PutStr("pipeline.name", buildRun.Definition.Name)
	attrs.PutStr("pipeline.run.id", fmt.Sprintf("%d", buildRun.ID))
	attrs.PutStr("pipeline.run.number", buildRun.BuildNumber)
	attrs.PutStr("pipeline.run.url", buildRun.URL)
	
	// Stage information (2 fields)
	attrs.PutStr("stage.id", job.ParentID)
	attrs.PutStr("stage.name", job.ParentID)
	
	// Job status and result (3 fields)
	attrs.PutStr("job.state", job.State)
	attrs.PutStr("job.result", job.Result)
	attrs.PutStr("status", mapJobResult(job.Result))
	
	// Timing information (5 fields)
	attrs.PutStr("job.start_time", job.StartTime.Format(time.RFC3339))
	attrs.PutStr("job.end_time", job.FinishTime.Format(time.RFC3339))
	attrs.PutDouble("job.duration_seconds", duration.Seconds())
	attrs.PutDouble("job.queue_time_seconds", queueDuration.Seconds())
	
	// VCS information (5 fields)
	attrs.PutStr("vcs.ref.name", buildRun.SourceBranch)
	attrs.PutStr("vcs.ref.revision", buildRun.SourceVersion)
	attrs.PutStr("vcs.repository.name", buildRun.Repository.Name)
	attrs.PutStr("vcs.repository.id", buildRun.Repository.ID)
	attrs.PutStr("vcs.repository.url", buildRun.Repository.URL)
	
	// Trigger information (3 fields)
	attrs.PutStr("pipeline.triggered_by", buildRun.RequestedFor.DisplayName)
	attrs.PutStr("pipeline.triggered_by.email", buildRun.RequestedFor.UniqueName)
	attrs.PutStr("pipeline.trigger_reason", buildRun.Reason)
	
	// Worker and metrics (4 fields)
	attrs.PutStr("job.worker_name", job.WorkerName)
	attrs.PutInt("job.error_count", int64(job.ErrorCount))
	attrs.PutInt("job.warning_count", int64(job.WarningCount))
	attrs.PutInt("job.order", int64(job.Order))
}
```

**Total Attributes**: 25+ fields covering all requirements from spec

#### 5.4 Severity Mapping

```go
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
```

**Mapping**:
- ✅ INFO: succeeded, skipped
- ✅ WARN: succeededWithIssues, canceled, abandoned
- ✅ ERROR: failed
- ✅ UNSPECIFIED: unknown results

#### 5.5 Status Normalization

```go
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
```

**Purpose**: Normalizes Azure DevOps job results to consistent status values

### 6. Build Verification

```bash
$ cd receiver/azuredevopsreceiver && go build
# Exit code: 0 ✅
```

**Result**: All code compiles successfully with no errors

### 7. Git Commit

```bash
$ git log --oneline -1
cd1cfe6 (HEAD -> main) feat: add OpenTelemetry logs support to Azure DevOps receiver
```

**Commit Message**:
```
feat: add OpenTelemetry logs support to Azure DevOps receiver

- Added LogsStability constant to metadata
- Extended ScraperFactory interface with CreateLogsScraper
- Implemented createLogsReceiver in factory
- Created logs.go with log record creation infrastructure
- Implemented createLogRecordFromJob with 25+ attributes
- Added severity mapping and resource attributes
- Logs infrastructure ready for pipeline metrics in Task 4.0

Related to T3.0 in Spec 01
```

## Implementation Summary

### Files Modified/Created

1. **receiver/azuredevopsreceiver/internal/metadata/generated_status.go**
   - Added `LogsStability` constant

2. **receiver/azuredevopsreceiver/internal/scraper.go**
   - Extended `ScraperFactory` interface with `CreateLogsScraper` method

3. **receiver/azuredevopsreceiver/factory.go**
   - Added `receiver.WithLogs` to factory
   - Implemented `createLogsReceiver` function

4. **receiver/azuredevopsreceiver/internal/scraper/azuredevopsscraper/factory.go**
   - Implemented `CreateLogsScraper` method

5. **receiver/azuredevopsreceiver/internal/scraper/azuredevopsscraper/logs.go** ⭐ NEW
   - 178 lines of log creation infrastructure
   - 5 functions: scrapeLogs, createLogRecordFromJob, jobToLogAttributes, mapJobResultToSeverity, mapJobResult

6. **docs/specs/01-spec-azure-devops-pipeline-metrics/01-tasks-azure-devops-pipeline-metrics.md**
   - Marked all Task 3.0 sub-tasks as complete

### Key Features Delivered

✅ **Logs Receiver Support**: Factory creates logs receivers via scraperhelper  
✅ **Resource Attributes**: VCS provider, owner, project identification  
✅ **Scope Definition**: azuredevops.pipeline.job scope with version  
✅ **Comprehensive Attributes**: 25+ fields mapped from job and build data  
✅ **Severity Mapping**: INFO/WARN/ERROR based on job results  
✅ **Timestamp Handling**: Proper timestamp and observed timestamp  
✅ **Status Normalization**: Consistent status values  
✅ **Build Success**: All code compiles without errors  

### Architecture Alignment

The implementation follows OpenTelemetry Collector best practices:

- Uses `plog` package for log data structures
- Follows resource/scope/log record hierarchy
- Implements proper timestamp handling
- Maps semantic conventions for VCS and pipeline attributes
- Integrates with existing scraper infrastructure
- Ready for pipeline data integration in Task 4.0

## Next Steps

Task 3.0 provides the complete logs infrastructure. Task 4.0 will:

1. Implement pipeline scraping logic in `scrapeLogs()`
2. Call `fetchBuildRuns()` and `fetchBuildTimeline()` from Task 2.0
3. Use `createLogRecordFromJob()` to emit logs for each pipeline job
4. Add configuration for pipeline filtering
5. Test end-to-end log emission to OpenSearch

## Conclusion

✅ **Task 3.0 Complete**: OpenTelemetry logs support successfully added to Azure DevOps receiver with comprehensive attribute mapping, severity handling, and resource identification. Infrastructure ready for pipeline metrics implementation.
