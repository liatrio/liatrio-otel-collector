# Task List: Azure DevOps Pipeline Metrics

**Spec Reference**: `01-spec-azure-devops-pipeline-metrics.md`  
**Created**: 2026-02-19  
**Status**: Parent Tasks Defined

---

## Overview

This task list implements the Azure DevOps Pipeline Metrics feature, which extends the existing Azure DevOps receiver to scrape pipeline run data and emit it as OpenTelemetry logs for visualization in Grafana via OpenSearch.

**Key Implementation Strategy**:
- Leverage existing scraper patterns from `deployments.go` and `work_items.go`
- Add new Builds API integration for YAML pipelines
- Introduce logs signal support to the receiver (new capability)
- Create local development environment with Docker Compose
- Implement job-level granularity with comprehensive field coverage

**Demoable Units**: Each parent task represents an end-to-end vertical slice that can be demonstrated independently.

---

## Tasks

### [x] 1.0 Create Local Development Environment with Docker Compose

Set up a complete local development stack with OpenSearch, Grafana, Prometheus, and the OTEL collector configured to send pipeline logs to OpenSearch.

#### 1.0 Proof Artifact(s)

- **Docker Compose**: `docker-compose up` successfully starts all services (OpenSearch, Grafana, Prometheus)
- **URL**: http://localhost:3000 shows Grafana login page
- **URL**: http://localhost:9200 shows OpenSearch cluster health
- **URL**: http://localhost:9090 shows Prometheus UI
- **Grafana Screenshot**: Grafana datasource configuration page shows OpenSearch datasource connected
- **OpenSearch Query**: `curl localhost:9200/otel-logs/_search` returns pipeline job log documents
- **README**: `docs/local-development.md` contains setup instructions

#### 1.0 Tasks

- [x] 1.1 Create `docker-compose.yml` in project root with OpenSearch, Grafana, and Prometheus services
  - OpenSearch service with ports 9200:9200 and 9600:9600
  - OpenSearch Dashboards service with port 5601:5601
  - Grafana service with port 3000:3000
  - Prometheus service with port 9090:9090
  - Configure volume mounts for persistence and configuration
  - Set environment variables for initial setup (disable security for local dev)
  
- [x] 1.2 Create OTEL collector configuration for local development
  - Create `config/config-local-dev.yaml` with logs pipeline
  - Configure OTLP receiver for logs
  - Configure OpenSearch exporter for logs (endpoint: http://opensearch:9200)
  - Configure Prometheus exporter for metrics
  - Add debug exporter for troubleshooting
  
- [x] 1.3 Create Grafana provisioning configuration
  - Create `grafana/provisioning/datasources/opensearch.yaml` for OpenSearch datasource
  - Create `grafana/provisioning/datasources/prometheus.yaml` for Prometheus datasource
  - Configure datasources to connect to local services
  
- [x] 1.4 Create local development documentation
  - Create `docs/local-development.md` with setup instructions
  - Document how to start the stack: `docker-compose up -d`
  - Document how to run the collector locally: `make run CONFIG=config/config-local-dev.yaml`
  - Document how to access services (URLs and default credentials)
  - Document how to verify logs are flowing to OpenSearch
  
- [x] 1.5 Test the complete stack
  - Run `docker-compose up -d` and verify all services start
  - Access Grafana at http://localhost:3000 (default admin/admin)
  - Access OpenSearch at http://localhost:9200
  - Access Prometheus at http://localhost:9090
  - Verify Grafana datasources are connected

---

### [ ] 2.0 Implement Builds API Integration and Data Structures

Create the foundational API integration for Azure DevOps Builds API to fetch pipeline runs and job details. This task establishes the data models and API client methods needed for all subsequent work.

#### 2.0 Proof Artifact(s)

- **Unit Test**: `TestFetchBuildRuns` passes, demonstrating successful API response parsing
- **Unit Test**: `TestFetchBuildTimeline` passes, demonstrating job extraction from timeline API
- **Unit Test**: `TestBuildStatusMapping` passes, demonstrating correct status normalization
- **CLI Output**: Running collector with debug logging shows "Fetched N build runs from Builds API" message
- **Code Review**: New file `receiver/azuredevopsreceiver/internal/scraper/azuredevopsscraper/builds.go` contains complete API integration

#### 2.0 Tasks

- [ ] 2.1 Create data structures for Builds API responses
  - Create `BuildRun` struct in `receiver/azuredevopsreceiver/internal/scraper/azuredevopsscraper/builds.go`
  - Create `BuildTimeline` struct for job/task details
  - Create `BuildTimelineRecord` struct for individual jobs
  - Add JSON tags matching Azure DevOps API response format
  - Use `NullableTime` for timestamp fields (follow existing pattern from deployments.go)
  
- [ ] 2.2 Implement `fetchBuildRuns` method
  - Add method to fetch builds from Azure DevOps Builds API
  - Endpoint: `GET https://dev.azure.com/{org}/{project}/_apis/build/builds`
  - Query parameters: `minTime`, `statusFilter=completed`, `$top=100`
  - Implement pagination using continuation tokens
  - Add error handling and retry logic for rate limiting
  
- [ ] 2.3 Implement `fetchBuildTimeline` method
  - Add method to fetch job details for a specific build
  - Endpoint: `GET https://dev.azure.com/{org}/{project}/_apis/build/builds/{buildId}/timeline`
  - Parse timeline records to extract job information
  - Filter for job-type records (exclude task-level details)
  
- [ ] 2.4 Implement status mapping helper
  - Create `mapBuildStatus` function to normalize Azure DevOps statuses
  - Map: `succeeded` → succeeded, `failed`/`partiallySucceeded` → failed, `canceled` → canceled
  - Handle edge cases and unknown statuses
  
- [ ] 2.5 Add unit tests for API integration
  - Create `builds_test.go` with test cases
  - Test `TestFetchBuildRuns` with mock API responses
  - Test `TestFetchBuildTimeline` with sample timeline data
  - Test `TestBuildStatusMapping` with various status values
  - Use testdata fixtures for realistic API responses

---

### [ ] 3.0 Add OpenTelemetry Logs Support to Azure DevOps Receiver

Extend the receiver factory to support the logs signal and implement log record creation from pipeline job data. This enables the receiver to emit logs alongside existing metrics and traces.

#### 3.0 Proof Artifact(s)

- **Unit Test**: `TestCreateLogRecordFromJob` passes, demonstrating log record creation with all required fields
- **Unit Test**: `TestJobToLogAttributes` passes, demonstrating correct attribute mapping
- **Integration Test**: Receiver factory creates logs receiver without errors
- **CLI Output**: `otelcol validate --config=config.yaml` passes with logs pipeline configured
- **Log Sample**: Example log record JSON in test output shows all 17+ required fields (job_name, pipeline_name, duration_seconds, etc.)

#### 3.0 Tasks

- [ ] 3.1 Add logs receiver support to factory
  - Modify `receiver/azuredevopsreceiver/factory.go`
  - Add `receiver.WithLogs(createLogsReceiver, metadata.LogsStability)` to factory
  - Implement `createLogsReceiver` function following existing patterns
  - Update `internal/metadata/generated_status.go` if needed for logs stability
  
- [ ] 3.2 Create log record builder for pipeline jobs
  - Create `receiver/azuredevopsreceiver/internal/scraper/azuredevopsscraper/logs.go`
  - Implement `createLogRecordFromJob` function
  - Use `plog.NewLogs()` and `plog.LogRecord` for log creation
  - Set log body to descriptive message (e.g., "Pipeline job completed")
  
- [ ] 3.3 Implement job-to-log attribute mapping
  - Create `jobToLogAttributes` helper function
  - Map all 17+ required fields to log attributes:
    - job_name, pipeline_name, pipeline_id, pipeline_run_id, stage_name
    - branch, duration_seconds, attempt, status
    - start_time, end_time, queue_time_seconds
    - triggered_by, trigger_reason
    - repository_name, repository_url, commit_sha
  - Set log severity based on status (INFO/ERROR/WARN)
  
- [ ] 3.4 Add resource attributes for logs
  - Set resource attributes: vcs.provider.name, vcs.owner.name, azuredevops.project
  - Follow existing resource builder pattern from metrics scraper
  
- [ ] 3.5 Create unit tests for log creation
  - Create `logs_test.go` with test cases
  - Test `TestCreateLogRecordFromJob` with sample job data
  - Test `TestJobToLogAttributes` verifies all fields are mapped
  - Verify log record structure matches expected format
  - Test severity mapping for different job statuses

---

### [ ] 4.0 Implement Pipeline Metrics Scraper with Configuration

Create the main scraper logic that orchestrates fetching pipeline runs, extracting jobs, and emitting log records. Add configuration parameters to control the feature.

#### 4.0 Proof Artifact(s)

- **Configuration File**: `config/config.yaml` contains new pipeline metrics configuration section
- **CLI Output**: Collector starts successfully with `pipeline_metrics_enabled: true` and logs "Pipeline metrics scraper initialized"
- **Unit Test**: `TestScrapePipelineMetrics` passes, demonstrating end-to-end scraping flow
- **Collector Logs**: Debug logs show "Scraped N pipeline runs, emitted M job log records" after scrape cycle
- **OpenTelemetry Logs**: Collector emits plog.Logs with job-level records (verified via debug exporter)

#### 4.0 Tasks

- [ ] 4.1 Add pipeline metrics configuration parameters
  - Modify `receiver/azuredevopsreceiver/internal/scraper/azuredevopsscraper/config.go`
  - Add `PipelineMetricsEnabled bool` field
  - Add `PipelineLookbackDays int` field (default 30)
  - Add `PipelineCollectionInterval time.Duration` field
  - Add `IncludeBuildPipelines bool` field (default true)
  - Add `IncludeReleasePipelines bool` field (default true)
  - Add mapstructure tags for YAML configuration
  
- [ ] 4.2 Implement main pipeline scraping orchestration
  - Modify `receiver/azuredevopsreceiver/internal/scraper/azuredevopsscraper/azuredevops_scraper.go`
  - Add `scrapePipelineMetrics` method to azuredevopsScraper struct
  - Calculate lookback time window based on `PipelineLookbackDays`
  - Call `fetchBuildRuns` to get pipeline runs
  - For each run, call `fetchBuildTimeline` to get jobs
  - Extract jobs and create log records
  
- [ ] 4.3 Integrate pipeline scraping into main scrape method
  - Modify the main `scrape` method in `azuredevops_scraper.go`
  - Add conditional check for `PipelineMetricsEnabled`
  - Call `scrapePipelineMetrics` if enabled
  - Handle errors gracefully without failing entire scrape
  - Add logging for scrape progress and results
  
- [ ] 4.4 Implement logs emission
  - Create method to emit plog.Logs from job records
  - Batch job log records into a single plog.Logs object
  - Return logs from scraper for consumption by logs pipeline
  - Handle empty results gracefully
  
- [ ] 4.5 Update example configuration
  - Modify `config/config.yaml` to add pipeline metrics section
  - Add commented example configuration with all new parameters
  - Document parameter meanings and recommended values
  
- [ ] 4.6 Add unit tests for scraper orchestration
  - Create tests in `azuredevops_scraper_test.go`
  - Test `TestScrapePipelineMetrics` with mock API responses
  - Test configuration parsing and validation
  - Test error handling for API failures
  - Verify log records are created correctly

---

### [ ] 5.0 Implement Grafana Dashboard for Pipeline Metrics

Create a Grafana dashboard that visualizes pipeline job metrics including duration trends, success rates, and job-level performance analysis.

#### 5.0 Proof Artifact(s)

- **Grafana Screenshot**: Dashboard showing "Job Duration Over Time" panel with trend line
- **Grafana Screenshot**: Dashboard showing "Pipeline Success Rate" panel with percentage
- **Grafana Screenshot**: Dashboard showing "Top 10 Slowest Jobs" table
- **Dashboard JSON**: `grafana/dashboards/pipeline-metrics.json` contains complete dashboard definition
- **URL**: http://localhost:3000/d/pipeline-metrics shows live dashboard with data
- **Query Example**: Dashboard panel queries documented in README showing OpenSearch query syntax

#### 5.0 Tasks

- [ ] 5.1 Create base dashboard JSON structure
  - Create `grafana/dashboards/pipeline-metrics.json`
  - Set up dashboard metadata (title, tags, timezone)
  - Configure dashboard variables for filtering (pipeline name, branch, etc.)
  - Set time range picker configuration
  
- [ ] 5.2 Create "Job Duration Over Time" panel
  - Add time series panel showing job duration trends
  - OpenSearch query: aggregate duration_seconds by job_name over time
  - Configure visualization: line chart with multiple series
  - Add thresholds for slow jobs (e.g., >300 seconds)
  
- [ ] 5.3 Create "Pipeline Success Rate" panel
  - Add stat panel showing percentage of successful jobs
  - OpenSearch query: calculate ratio of succeeded vs total jobs
  - Configure visualization: percentage gauge or stat
  - Color coding: green for >90%, yellow for 70-90%, red for <70%
  
- [ ] 5.4 Create "Top 10 Slowest Jobs" panel
  - Add table panel showing jobs with highest average duration
  - OpenSearch query: aggregate avg(duration_seconds) grouped by job_name
  - Sort by duration descending, limit 10
  - Include columns: job_name, pipeline_name, avg_duration, last_run
  
- [ ] 5.5 Create "Job Status Distribution" panel
  - Add pie chart or bar chart showing job status breakdown
  - OpenSearch query: count jobs grouped by status
  - Show succeeded, failed, canceled counts
  
- [ ] 5.6 Add dashboard to Grafana provisioning
  - Create `grafana/provisioning/dashboards/dashboards.yaml`
  - Configure dashboard provider to load from `grafana/dashboards/` directory
  - Update docker-compose.yml to mount dashboard files
  
- [ ] 5.7 Test dashboard with sample data
  - Verify dashboard loads in Grafana
  - Test all panels render correctly
  - Test variable filters work as expected
  - Document example queries in README

---

### [ ] 6.0 Add Integration Tests and Documentation

Implement comprehensive integration tests and update documentation to guide users through configuration and usage.

#### 6.0 Proof Artifact(s)

- **Integration Test**: `TestPipelineMetricsEndToEnd` passes, demonstrating full scrape → logs → export flow
- **Test Output**: `make test` passes with pipeline metrics tests included
- **Documentation**: `receiver/azuredevopsreceiver/README.md` updated with pipeline metrics section
- **Configuration Example**: README contains complete example config with all pipeline metrics parameters
- **CLI Output**: `make checks` passes (linting, formatting, tests)
- **Code Coverage**: Pipeline metrics code has >80% test coverage (verified via codecov or go test -cover)

#### 6.0 Tasks

- [ ] 6.1 Create integration test for end-to-end flow
  - Create `receiver/azuredevopsreceiver/integration_test.go`
  - Test `TestPipelineMetricsEndToEnd` with mock Azure DevOps API
  - Verify scraper fetches builds, extracts jobs, creates logs
  - Verify logs are emitted with correct structure and fields
  - Use httptest for mocking API responses
  
- [ ] 6.2 Add tests for configuration validation
  - Test invalid configuration values are rejected
  - Test default values are applied correctly
  - Test configuration unmarshaling from YAML
  
- [ ] 6.3 Update receiver README documentation
  - Modify `receiver/azuredevopsreceiver/README.md`
  - Add "Pipeline Metrics" section after existing sections
  - Document new configuration parameters with examples
  - Document required Azure DevOps permissions (Build Read)
  - Add example OpenSearch queries for common use cases
  
- [ ] 6.4 Create configuration examples
  - Add complete example config to README
  - Document recommended settings for different project sizes
  - Document API rate limit considerations
  - Add troubleshooting section for common issues
  
- [ ] 6.5 Run quality checks
  - Run `make checks` to verify linting passes
  - Run `make test` to verify all tests pass
  - Check code coverage for new code (target >80%)
  - Fix any linting or formatting issues
  
- [ ] 6.6 Update metadata and generated files if needed
  - Run metadata generation if new metrics/attributes added
  - Update `internal/metadata/` files if schema changed
  - Verify generated code compiles and tests pass

---

## Relevant Files

### Files to Create

**Infrastructure & Configuration:**
- `docker-compose.yml` - Docker Compose stack for local development
- `config/config-local-dev.yaml` - OTEL collector config for local dev
- `grafana/provisioning/datasources/opensearch.yaml` - OpenSearch datasource config
- `grafana/provisioning/datasources/prometheus.yaml` - Prometheus datasource config
- `grafana/provisioning/dashboards/dashboards.yaml` - Dashboard provider config
- `grafana/dashboards/pipeline-metrics.json` - Pipeline metrics dashboard
- `docs/local-development.md` - Local development setup guide

**Go Source Files:**
- `receiver/azuredevopsreceiver/internal/scraper/azuredevopsscraper/builds.go` - Builds API integration
- `receiver/azuredevopsreceiver/internal/scraper/azuredevopsscraper/builds_test.go` - Builds API tests
- `receiver/azuredevopsreceiver/internal/scraper/azuredevopsscraper/logs.go` - Log record creation
- `receiver/azuredevopsreceiver/internal/scraper/azuredevopsscraper/logs_test.go` - Log creation tests
- `receiver/azuredevopsreceiver/integration_test.go` - End-to-end integration tests

### Files to Modify

**Configuration:**
- `config/config.yaml` - Add pipeline metrics configuration example

**Go Source Files:**
- `receiver/azuredevopsreceiver/factory.go` - Add logs receiver support
- `receiver/azuredevopsreceiver/internal/scraper/azuredevopsscraper/config.go` - Add pipeline config fields
- `receiver/azuredevopsreceiver/internal/scraper/azuredevopsscraper/azuredevops_scraper.go` - Add pipeline scraping logic
- `receiver/azuredevopsreceiver/internal/scraper/azuredevopsscraper/azuredevops_scraper_test.go` - Add scraper tests

**Documentation:**
- `receiver/azuredevopsreceiver/README.md` - Add pipeline metrics documentation

**Metadata (if needed):**
- `receiver/azuredevopsreceiver/internal/metadata/generated_status.go` - May need logs stability constant

### Test Data Files (to create)
- `receiver/azuredevopsreceiver/internal/scraper/azuredevopsscraper/testdata/build-run-response.json` - Sample build API response
- `receiver/azuredevopsreceiver/internal/scraper/azuredevopsscraper/testdata/build-timeline-response.json` - Sample timeline API response

---

## Dependencies

### External Dependencies
- Azure DevOps Builds API (v7.1+)
- OpenTelemetry Collector plog package
- Docker & Docker Compose (for local dev)
- OpenSearch (latest stable)
- Grafana (latest stable)
- Prometheus (latest stable)

### Internal Dependencies
- Existing Azure DevOps receiver framework
- Existing scraper patterns (deployments.go, work_items.go)
- Existing authentication extensions (bearertokenauth)
- Existing metadata builder patterns

---

## Implementation Notes

### Architectural Decisions

1. **Logs vs Metrics**: Using OpenTelemetry logs (not metrics) per user requirement for OpenSearch ingestion
2. **Job-Level Granularity**: Each job emitted as separate log record for detailed analysis in Grafana
3. **API Strategy**: Builds API for YAML pipelines, Release API for release pipelines (dual support)
4. **Scraper Pattern**: Follow existing pattern from deployments.go for consistency
5. **Configuration**: Separate `pipeline_collection_interval` to manage API rate limits independently

### Key Technical Considerations

- **API Rate Limits**: Expected 200-300 API calls per scrape for 100 pipelines/day
- **Lookback Window**: Default 30 days, configurable via `pipeline_lookback_days`
- **Status Filtering**: Only completed runs (succeeded, failed, canceled) - no in-progress
- **Field Coverage**: 17+ fields per job including timestamps, trigger info, repository context
- **Error Handling**: Graceful handling of missing fields using NullableTime pattern

### Testing Strategy

- Unit tests for API parsing, data transformation, log creation
- Integration tests for end-to-end scraping flow
- Local environment testing with real OpenSearch/Grafana
- Mock Azure DevOps API responses for deterministic tests

---

## Timeline

**Target**: Working in dev environment today (2026-02-19)

**Recommended Sequence**:
1. Task 1.0 (Local dev environment) - ~1 hour - **START HERE**
2. Tasks 2.0 + 3.0 (Core functionality) - ~2-3 hours
3. Task 4.0 (Integration) - ~1-2 hours  
4. Task 5.0 (Dashboard) - ~1 hour
5. Task 6.0 (Tests & docs) - ~1-2 hours

**Critical Path**: 1.0 → 2.0 → 3.0 → 4.0 (minimum viable for local testing)

---

## Next Steps

**Awaiting User Confirmation**: Please review the parent tasks above and confirm they align with your expectations.

Once confirmed, respond with **"Generate sub tasks"** and I will break down each parent task into detailed implementation steps with specific file changes.
