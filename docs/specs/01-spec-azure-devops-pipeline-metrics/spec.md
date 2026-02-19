# Specification: Azure DevOps Pipeline Metrics

**Spec ID**: 01  
**Feature Name**: Azure DevOps Pipeline Metrics  
**Status**: Draft  
**Created**: 2026-02-19  
**Last Updated**: 2026-02-19  
**Priority**: High (needed in dev today)

---

## 1. Overview

### 1.1 Purpose

Extend the liatrio-otel-collector's Azure DevOps receiver to scrape pipeline run metrics and emit them as OpenTelemetry logs for ingestion into OpenSearch and visualization in Grafana. This will provide visibility into pipeline execution patterns, job-level performance, and trends over time.

### 1.2 Background

The Azure DevOps receiver currently supports:
- VCS metrics (repositories, branches, pull requests, code coverage)
- Deployment metrics via Release Management API
- Work item metrics
- Webhook-based trace receiver for pipeline events

This spec adds **pipeline run metrics scraping** to complement the existing capabilities, supporting both:
- **Release Pipelines** (via Release Management API - existing pattern)
- **Build Pipelines** (via Builds API - new capability for YAML pipelines)

### 1.3 Goals

- Scrape all pipelines in the configured Azure DevOps project
- Capture pipeline, stage, and job-level metrics with comprehensive field coverage
- Emit data as OpenTelemetry logs for OpenSearch ingestion
- Support configurable historical lookback (default 30 days)
- Enable Grafana dashboards showing job duration trends and performance analysis
- Provide local development environment setup with Docker Compose

### 1.4 Non-Goals

- Real-time pipeline monitoring (scraper-based, not webhook-based)
- Environment/target tracking (not needed per user requirement)
- In-progress pipeline runs (only completed runs)
- Custom retry tracking beyond Azure DevOps attempt numbers

---

## 2. User Stories

### Story 1: Pipeline Metrics Collection
**As a** DevOps engineer  
**I want** the OTEL collector to scrape all pipeline runs from my Azure DevOps project  
**So that** I can analyze pipeline execution patterns and performance in Grafana

**Acceptance Criteria**:
- Collector scrapes all pipelines in the configured project
- Both Build pipelines (YAML) and Release pipelines are supported
- Pipeline runs from the last 30 days (configurable) are collected
- Only completed runs (succeeded/failed/canceled) are included

**Proof Artifacts**:
- OTEL collector logs showing successful pipeline scraping
- OpenSearch index containing pipeline run documents
- Grafana query returning pipeline run data

### Story 2: Job-Level Granularity
**As a** DevOps engineer  
**I want** job-level metrics for each pipeline run  
**So that** I can identify which specific jobs are slow or failing

**Acceptance Criteria**:
- Each job within a pipeline run is emitted as a separate log entry
- Job name, duration, status, and attempt number are captured
- Pipeline and stage context is included with each job entry
- All fields specified in section 3.2 are present

**Proof Artifacts**:
- OpenSearch query showing individual job documents
- Grafana dashboard displaying job duration trends over time
- Sample log entry with all required fields

### Story 3: Local Development Environment
**As a** developer  
**I want** a Docker Compose setup for local development  
**So that** I can quickly iterate on the collector with OpenSearch, Grafana, and Prometheus running locally

**Acceptance Criteria**:
- Docker Compose file spins up OpenSearch, Grafana, Prometheus, and dependencies
- OTEL collector configuration sends logs to OpenSearch
- Grafana is pre-configured to query OpenSearch
- README or documentation explains how to run the stack

**Proof Artifacts**:
- `docker-compose up` successfully starts all services
- Grafana UI accessible at http://localhost:3000
- OpenSearch receiving logs from OTEL collector
- Example Grafana dashboard showing pipeline metrics

---

## 3. Functional Requirements

### 3.1 Pipeline Data Sources

**FR-1.1**: The scraper SHALL support two Azure DevOps APIs:
- **Builds API** for YAML pipelines: `https://dev.azure.com/{org}/{project}/_apis/build/builds`
- **Release Management API** for Release pipelines: `https://vsrm.dev.azure.com/{org}/{project}/_apis/release/deployments`

**FR-1.2**: The scraper SHALL fetch all pipelines in the configured project (no filtering by name or pattern)

**FR-1.3**: The scraper SHALL use API version 7.1 or compatible

### 3.2 Data Fields

**FR-2.1**: Each job log entry SHALL include the following fields:

**Core Identification**:
- `job_name` - Name of the job
- `pipeline_name` - Name of the pipeline
- `pipeline_id` - Unique pipeline identifier
- `pipeline_run_id` - Unique run identifier
- `stage_name` - Name of the stage containing the job

**Execution Details**:
- `branch` - Branch name that triggered the run
- `duration_seconds` - Job execution duration in seconds
- `attempt` - Attempt/retry number (1 for first attempt, 2+ for retries)
- `status` - Job status (succeeded, failed, canceled, etc.)

**Timestamps**:
- `start_time` - Job start timestamp (ISO 8601)
- `end_time` - Job end timestamp (ISO 8601)
- `queue_time_seconds` - Time spent waiting in queue before execution

**Trigger Information**:
- `triggered_by` - User or service principal that triggered the run
- `trigger_reason` - Reason for trigger (manual, CI, scheduled, etc.)

**Repository Context**:
- `repository_name` - Repository name
- `repository_url` - Repository URL
- `commit_sha` - Git commit SHA

**FR-2.2**: Pipeline-level and stage-level context SHALL be included with each job entry to enable grouping in queries

**FR-2.3**: All timestamp fields SHALL use ISO 8601 format with timezone information

### 3.3 Data Output Format

**FR-3.1**: Pipeline metrics SHALL be emitted as **OpenTelemetry Logs** (structured log records)

**FR-3.2**: Each job SHALL be emitted as a separate log entry (one document per job in OpenSearch)

**FR-3.3**: Log entries SHALL include resource attributes identifying the Azure DevOps organization and project

**FR-3.4**: Log severity SHALL be set based on job status:
- `INFO` for succeeded jobs
- `ERROR` for failed jobs
- `WARN` for canceled jobs

### 3.4 Historical Data Collection

**FR-4.1**: The scraper SHALL support a configurable lookback window (config parameter: `pipeline_lookback_days`)

**FR-4.2**: Default lookback SHALL be 30 days if not configured

**FR-4.3**: On each scrape cycle, the scraper SHALL fetch all completed runs within the lookback window

**FR-4.4**: Only completed runs SHALL be included (states: succeeded, failed, canceled)

**FR-4.5**: In-progress or pending runs SHALL be excluded

### 3.5 Collection Interval

**FR-5.1**: Pipeline metrics scraping SHALL support a separate collection interval from other scrapers

**FR-5.2**: Configuration parameter `pipeline_collection_interval` SHALL control scraping frequency

**FR-5.3**: Default interval SHALL be 15 minutes (900 seconds) to balance freshness and API rate limits

**FR-5.4**: The scraper SHALL respect Azure DevOps API rate limits and implement appropriate backoff

### 3.6 Configuration

**FR-6.1**: The following configuration parameters SHALL be added to the Azure DevOps scraper config:

```yaml
azuredevops:
  scrapers:
    azuredevops:
      # Existing config...
      
      # Pipeline metrics configuration
      pipeline_metrics_enabled: true           # Enable/disable pipeline metrics
      pipeline_lookback_days: 30               # Days of history to fetch
      pipeline_collection_interval: 900s       # Separate interval for pipeline scraping
      include_build_pipelines: true            # Scrape YAML/Build pipelines
      include_release_pipelines: true          # Scrape Release pipelines
```

**FR-6.2**: Pipeline metrics SHALL be disabled by default (`pipeline_metrics_enabled: false`)

**FR-6.3**: Both build and release pipelines SHALL be enabled by default when pipeline metrics are enabled

---

## 4. Technical Considerations

### 4.1 API Integration

**TC-1.1**: Use Azure DevOps Builds API for YAML pipelines:
- Endpoint: `GET https://dev.azure.com/{organization}/{project}/_apis/build/builds`
- Query parameters: `minTime`, `maxTime`, `statusFilter=completed`
- Pagination via continuation tokens

**TC-1.2**: Use Azure DevOps Release Management API for Release pipelines:
- Endpoint: `GET https://vsrm.dev.azure.com/{organization}/{project}/_apis/release/deployments`
- Follow existing pattern from `deployments.go`

**TC-1.3**: For each pipeline run, fetch detailed job information:
- Build API: `GET https://dev.azure.com/{organization}/{project}/_apis/build/builds/{buildId}/timeline`
- Release API: Job details included in deployment response

**TC-1.4**: Implement retry logic with exponential backoff for API rate limiting (HTTP 429)

### 4.2 Data Modeling

**TC-2.1**: Create Go structs for API responses:
- `BuildRun` - Represents a build pipeline run
- `BuildTimeline` - Contains job/task details for a build
- `ReleaseRun` - Represents a release pipeline run (may reuse existing `Deployment` struct)

**TC-2.2**: Map Azure DevOps status values to normalized statuses:
- `succeeded` → succeeded
- `failed`, `partiallySucceeded` → failed
- `canceled` → canceled

**TC-2.3**: Handle missing or null fields gracefully (use `NullableTime` pattern from existing code)

### 4.3 OpenTelemetry Logs

**TC-3.1**: Use `plog` package for log record creation:
```go
import "go.opentelemetry.io/collector/pdata/plog"
```

**TC-3.2**: Structure log records with:
- Body: JSON-encoded job details
- Attributes: Structured key-value pairs for all fields
- Resource attributes: Organization, project, provider name
- Timestamp: Job end time
- Severity: Based on job status

**TC-3.3**: Create a new logs receiver method in the scraper or extend existing scraper to support logs signal

### 4.4 Performance & Rate Limiting

**TC-4.1**: Expected API call volume for 100 pipeline runs/day:
- ~100 calls to list builds
- ~100 calls to fetch timelines
- ~100 calls for release deployments (if applicable)
- **Total: ~200-300 API calls per scrape cycle**

**TC-4.2**: With 15-minute intervals (96 scrapes/day), this is manageable within Azure DevOps rate limits

**TC-4.3**: Implement concurrency control using existing `concurrency_limit` pattern

**TC-4.4**: Add metrics to track API call counts and rate limit errors

### 4.5 Local Development Environment

**TC-5.1**: Create `docker-compose.yml` with services:
- OpenSearch (latest stable)
- OpenSearch Dashboards (latest stable)
- Grafana (latest stable)
- Prometheus (latest stable)

**TC-5.2**: OTEL collector configuration:
- Logs receiver: OTLP
- Logs exporter: OpenSearch (via HTTP)
- Metrics exporter: Prometheus

**TC-5.3**: Grafana provisioning:
- OpenSearch datasource pre-configured
- Example dashboard for pipeline metrics

**TC-5.4**: Volume mounts for:
- OTEL collector config
- Grafana dashboards
- OpenSearch data persistence

---

## 5. Dependencies

### 5.1 External Dependencies

- Azure DevOps Builds API (v7.1+)
- Azure DevOps Release Management API (v7.1+)
- OpenTelemetry Collector (existing)
- OpenSearch (latest stable - for local dev)
- Grafana (latest stable - for local dev)
- Prometheus (latest stable - for local dev)

### 5.2 Internal Dependencies

- Existing Azure DevOps receiver (`receiver/azuredevopsreceiver`)
- Existing authentication extensions (bearer token auth)
- Existing scraper helper utilities
- Existing `NullableTime` and error handling patterns

### 5.3 Authentication

- Requires Azure DevOps PAT with **Build (Read)** permissions
- Optionally **Release (Read)** permissions for Release pipelines
- Uses existing `bearertokenauth/azuredevops` extension

---

## 6. Testing Strategy

### 6.1 Unit Tests

- Test API response parsing for Builds API
- Test API response parsing for Release API
- Test job extraction from timeline data
- Test status mapping logic
- Test log record creation with all fields
- Test error handling for missing/null fields
- Test pagination handling

### 6.2 Integration Tests

- Test scraping against Azure DevOps test project
- Verify log records contain all required fields
- Test with both build and release pipelines
- Test lookback window behavior
- Test rate limiting and retry logic

### 6.3 Local Development Testing

- Verify Docker Compose stack starts successfully
- Verify logs appear in OpenSearch
- Verify Grafana can query and display pipeline metrics
- Test collector configuration changes

---

## 7. Documentation Requirements

### 7.1 User Documentation

- Update `receiver/azuredevopsreceiver/README.md` with pipeline metrics section
- Document new configuration parameters
- Provide example configuration
- Document required Azure DevOps permissions

### 7.2 Developer Documentation

- Document API integration approach
- Document data model and structs
- Provide local development setup instructions
- Include example queries for OpenSearch/Grafana

### 7.3 Example Artifacts

- Example OTEL collector config with pipeline metrics enabled
- Example docker-compose.yml for local development
- Example Grafana dashboard JSON
- Example OpenSearch index mapping

---

## 8. Success Metrics

### 8.1 Functional Success

- [ ] Collector successfully scrapes all pipelines from configured project
- [ ] Job-level logs appear in OpenSearch with all required fields
- [ ] Grafana dashboard displays job duration trends
- [ ] Both build and release pipelines are supported
- [ ] Only completed runs are included

### 8.2 Performance Success

- [ ] Scrape completes within 5 minutes for 100 pipeline runs
- [ ] No Azure DevOps API rate limit errors under normal operation
- [ ] Memory usage remains stable across scrape cycles
- [ ] API call count is within expected range (200-300 per cycle)

### 8.3 Development Success

- [ ] Docker Compose stack starts in under 2 minutes
- [ ] Local development environment fully functional
- [ ] Developer can iterate on collector code and see results in Grafana
- [ ] All services accessible via localhost

---

## 9. Timeline & Milestones

**Priority**: High - needed in dev today (2026-02-19)

### Phase 1: Core Implementation (Today)
- [ ] Implement Builds API integration
- [ ] Implement job extraction and log record creation
- [ ] Add configuration parameters
- [ ] Basic unit tests

### Phase 2: Local Development Environment (Today)
- [ ] Create docker-compose.yml
- [ ] Configure OTEL collector for logs → OpenSearch
- [ ] Set up Grafana with OpenSearch datasource
- [ ] Create example dashboard

### Phase 3: Release Pipelines Support (Follow-up)
- [ ] Extend to Release Management API
- [ ] Unified job extraction for both pipeline types
- [ ] Integration tests

### Phase 4: Documentation & Polish (Follow-up)
- [ ] Update README
- [ ] Add comprehensive tests
- [ ] Performance optimization
- [ ] Production-ready configuration examples

---

## 10. Open Questions

1. **Q**: Should the scraper deduplicate runs if the same run appears in multiple scrape cycles?  
   **A**: TBD - likely no deduplication, rely on OpenSearch to handle duplicates by document ID

2. **Q**: How should we handle very large pipeline runs with 100+ jobs?  
   **A**: TBD - may need pagination or batching for log emission

3. **Q**: Should we emit pipeline-level and stage-level logs in addition to job-level?  
   **A**: TBD - current spec is job-level only, but could add aggregated entries

4. **Q**: What should be the unique identifier for log records to enable deduplication?  
   **A**: TBD - likely combination of `pipeline_run_id` + `job_name` + `attempt`

---

## 11. Risks & Mitigations

### Risk 1: API Rate Limiting
**Impact**: High  
**Probability**: Medium  
**Mitigation**: 
- Use separate collection interval (15 min default)
- Implement exponential backoff
- Monitor API call counts
- Document rate limit considerations

### Risk 2: Large Data Volume
**Impact**: Medium  
**Probability**: Medium  
**Mitigation**:
- Job-level granularity may create many log entries
- 100 pipelines/day × avg 10 jobs/pipeline = 1000 logs/day (manageable)
- Implement configurable retention in OpenSearch
- Consider sampling if volume becomes problematic

### Risk 3: Timeline Complexity
**Impact**: High  
**Probability**: High  
**Mitigation**:
- Aggressive timeline (today) may require phased delivery
- Prioritize core functionality (Builds API + logs) first
- Release pipelines and polish can follow
- Set clear expectations with stakeholders

### Risk 4: OpenTelemetry Logs Signal
**Impact**: Medium  
**Probability**: Low  
**Mitigation**:
- Logs signal is well-supported in OTEL collector
- OpenSearch exporter supports logs
- Follow existing OTEL patterns and documentation
- Test thoroughly in local environment first

---

## 12. Appendix

### 12.1 Example Configuration

```yaml
extensions:
  bearertokenauth/azuredevops:
    token: ${env:ADO_PAT}

receivers:
  azuredevops:
    initial_delay: 10s
    collection_interval: 1800s  # 30 minutes for VCS/deployment metrics
    scrapers:
      azuredevops:
        organization: liatrio
        project: flywheel
        base_url: "https://dev.azure.com"
        
        # Pipeline metrics configuration
        pipeline_metrics_enabled: true
        pipeline_lookback_days: 30
        pipeline_collection_interval: 900s  # 15 minutes for pipeline metrics
        include_build_pipelines: true
        include_release_pipelines: true
        
        auth:
          authenticator: bearertokenauth/azuredevops

exporters:
  opensearch:
    http:
      endpoint: http://localhost:9200
    logs_index: otel-logs
  
  prometheus:
    endpoint: "0.0.0.0:8889"

service:
  extensions: [bearertokenauth/azuredevops]
  pipelines:
    logs:
      receivers: [azuredevops]
      exporters: [opensearch]
    metrics:
      receivers: [azuredevops]
      exporters: [prometheus]
```

### 12.2 Example Log Record Structure

```json
{
  "timestamp": "2026-02-19T13:45:30Z",
  "severity": "INFO",
  "body": "Pipeline job completed",
  "attributes": {
    "job_name": "Build and Test",
    "pipeline_name": "CI Pipeline",
    "pipeline_id": "123",
    "pipeline_run_id": "5678",
    "stage_name": "Build",
    "branch": "main",
    "duration_seconds": 245,
    "attempt": 1,
    "status": "succeeded",
    "start_time": "2026-02-19T13:41:25Z",
    "end_time": "2026-02-19T13:45:30Z",
    "queue_time_seconds": 15,
    "triggered_by": "john.doe@example.com",
    "trigger_reason": "continuous integration",
    "repository_name": "my-app",
    "repository_url": "https://dev.azure.com/org/project/_git/my-app",
    "commit_sha": "abc123def456"
  },
  "resource": {
    "vcs.provider.name": "azuredevops",
    "vcs.owner.name": "liatrio",
    "azuredevops.project": "flywheel"
  }
}
```

### 12.3 Azure DevOps API References

- [Builds API](https://learn.microsoft.com/en-us/rest/api/azure/devops/build/builds)
- [Build Timeline API](https://learn.microsoft.com/en-us/rest/api/azure/devops/build/timeline)
- [Release Deployments API](https://learn.microsoft.com/en-us/rest/api/azure/devops/release/deployments)
- [API Rate Limits](https://learn.microsoft.com/en-us/azure/devops/integrate/concepts/rate-limits)

---

**End of Specification**
