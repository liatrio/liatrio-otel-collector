# 01 Questions Round 1 - Azure DevOps Pipeline Metrics

Please answer each question below (select one or more options, or add your own notes). Feel free to add additional context under any question.

## 1. Pipeline Metrics Data Source

Which Azure DevOps API should be used to fetch pipeline run metrics?

- [ ] (A) **Builds API** - Uses the Azure DevOps Builds API (https://dev.azure.com/{org}/{project}/_apis/build/builds) which provides pipeline runs, including YAML pipelines
- [ ] (B) **Pipelines Runs API** - Uses the newer Pipelines API (https://dev.azure.com/{org}/{project}/_apis/pipelines/{pipelineId}/runs) which is more modern but may have different data structure
- [ ] (C) **Release Management API** - Uses the existing Release Management API pattern (similar to current deployment metrics implementation)
- [ ] (D) Other (describe)

**Context**: The codebase currently uses the Release Management API for deployment metrics. The Builds API is commonly used for YAML pipeline runs and includes job-level details.

I need to be able to pull release pipelines (C), but also all regular pipelines (A).

## 2. Pipeline Identification & Filtering

How should the scraper identify which pipelines to collect metrics from?

- [ ] (A) **All pipelines in project** - Scrape all pipelines in the configured project
- [ ] (B) **Specific pipeline names** - Configure specific pipeline name(s) to scrape (similar to current `deployment_pipeline_name` config)
- [ ] (C) **Pipeline name pattern/filter** - Use a pattern or filter to match multiple pipelines (e.g., "prod-*")
- [ ] (D) **Pipeline tags/labels** - Filter pipelines based on Azure DevOps tags or labels
- [ ] (E) Other (describe)

Option A
## 3. Job-Level Metrics Granularity

What level of detail is needed for job metrics?

- [ ] (A) **Pipeline run summary only** - Overall pipeline duration, status, branch (no individual job details)
- [ ] (B) **Job-level metrics** - Individual job names, durations, statuses within each pipeline run
- [ ] (C) **Stage-level metrics** - Stage names, durations, statuses (grouping of jobs)
- [ ] (D) **All levels** - Pipeline, stage, and job-level metrics
- [ ] (E) Other (describe)

**Note**: More granularity means more API calls and potentially more data volume.

Option D

## 4. Environment/Target Information

How should the "targeted environment" be determined?

- [ ] (A) **From stage name** - Extract environment from the stage name (e.g., "Deploy to Production" → "Production")
- [ ] (B) **From pipeline variables** - Read from pipeline variables/parameters (e.g., `environment` variable)
- [ ] (C) **From branch name** - Infer from branch (e.g., `main` → "production", `develop` → "staging")
- [ ] (D) **Configuration mapping** - User-defined mapping in collector config (e.g., map specific pipelines to environments)
- [ ] (E) **Not needed** - Don't track environment information
- [ ] (F) Other (describe)

Option E

## 5. Retry Tracking

How should pipeline retries be tracked?

- [ ] (A) **Retry count per run** - Track how many times a specific run was retried (if Azure DevOps provides this)
- [ ] (B) **Attempt number** - Track the attempt number for each run (1st attempt, 2nd attempt, etc.)
- [ ] (C) **Separate runs** - Treat each retry as a separate pipeline run (no special retry tracking)
- [ ] (D) **Not needed** - Don't track retry information
- [ ] (E) Other (describe)

**Context**: Azure DevOps may expose retry/attempt information differently depending on the API used.

Option B
## 6. Data Output Format

You mentioned "feed in as logs in opensearch" - what format should the pipeline metrics use?

- [ ] (A) **OpenTelemetry Logs** - Emit as OTEL log records (structured logs with attributes)
- [ ] (B) **OpenTelemetry Metrics** - Continue using OTEL metrics (gauges, counters) like current implementation
- [ ] (C) **OpenTelemetry Traces** - Emit as trace spans (similar to existing webhook trace receiver)
- [ ] (D) **Hybrid approach** - Use logs for detailed pipeline run data, metrics for aggregations
- [ ] (E) Other (describe)

**Context**: The current Azure DevOps scraper emits OTEL metrics. The webhook receiver emits traces. Logs would be a new signal type for this receiver.

Option A

## 7. Historical Data Lookback

How far back should the scraper look for pipeline run history?

- [ ] (A) **Configurable lookback days** - Similar to `deployment_lookback_days` (default 30 days)
- [ ] (B) **Since last scrape** - Only fetch new runs since the last collection (requires state management)
- [ ] (C) **Fixed time window** - Always fetch last N days (e.g., always last 7 days)
- [ ] (D) **All recent runs** - Fetch all runs from a reasonable time period without configuration
- [ ] (E) Other (describe)

Option A

## 8. Collection Interval & Performance

Given the existing Azure DevOps API rate limiting concerns, what's the expected collection interval?

- [ ] (A) **Use existing interval** - Use the same collection_interval as other scrapers (currently recommended 15-60 min)
- [ ] (B) **Separate interval** - Configure a separate interval specifically for pipeline metrics
- [ ] (C) **On-demand only** - Don't scrape periodically, only collect via webhooks or manual trigger
- [ ] (D) Other (describe)

**Context**: The README warns about API rate limits with 540+ calls per scrape for 60 repos. Pipeline metrics will add more API calls.

Option B

## 9. Local Development Environment Setup

For your local development environment with Prometheus, Grafana, OpenSearch, and OTEL collector:

- [ ] (A) **Docker Compose provided** - Would you like a docker-compose.yml file to spin up the full stack?
- [ ] (B) **Configuration examples** - Need example config files for the collector to send logs to OpenSearch?
- [ ] (C) **Both** - Both docker-compose and configuration examples
- [ ] (D) **Already have setup** - You already have the environment, just need the collector changes

**Additional details needed**:
- OpenSearch version preference: _________
- Grafana version preference: _________
- Prometheus version preference: _________

Option A. Use latest stable versions of published docker images for opensearch, grafana, and prometheus
## 10. Pipeline Metrics Fields - Confirmation

You mentioned these fields. Please confirm or adjust:

**Confirmed fields**:
- [ ] Job name
- [ ] Pipeline name  
- [ ] Targeted environment
- [ ] Branch name
- [ ] Duration (in seconds?)
- [ ] Retry count/attempt
- [ ] Status (succeeded, failed, canceled, etc.)

**Additional fields to consider**:
- [ ] Pipeline run ID
- [ ] Triggered by (user/service principal)
- [ ] Trigger reason (manual, CI, scheduled, etc.)
- [ ] Repository name/URL
- [ ] Commit SHA
- [ ] Start time / End time (timestamps)
- [ ] Queue time (time waiting before execution)

**Fields NOT needed**:
- (List any fields you don't need)

Please include all of these fields

## 11. OpenSearch Index Structure

How should the data be structured in OpenSearch?

- [ ] (A) **One document per pipeline run** - Each pipeline run is a single log entry with nested job details
- [ ] (B) **One document per job** - Each job within a pipeline run is a separate log entry
- [ ] (C) **Separate indices** - Different indices for pipeline-level vs job-level data
- [ ] (D) **Let OTEL exporter decide** - Use default OTEL logs exporter behavior
- [ ] (E) Other (describe)

Option B I think is the best because I want to display on the dashboard how long each job is taking and their trends over time.

## 12. Error Handling & Incomplete Runs

How should the scraper handle in-progress or failed pipeline runs?

- [ ] (A) **Include all states** - Scrape and emit data for in-progress, failed, succeeded, and canceled runs
- [ ] (B) **Completed only** - Only emit data for completed runs (succeeded or failed)
- [ ] (C) **Configurable filter** - Allow configuration to filter by run state
- [ ] (D) Other (describe)

Option B
---

## Additional Questions or Context

Please provide any additional context, requirements, or constraints:

- **Specific Azure DevOps version/hosting**: (Cloud, Server, version?)
- **Expected data volume**: (How many pipelines? How many runs per day?)
- **Specific use cases**: (What will you do with this data in OpenSearch/Grafana?)
- **Timeline/Priority**: (When do you need this feature?)
- **Any other requirements**:

This data will all be displayed in grafana. I would like to get this specific functionality working in dev today. I expect about 100 pipelines a day. 
