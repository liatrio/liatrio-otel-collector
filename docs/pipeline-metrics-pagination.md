# Pipeline Metrics Pagination Configuration

## Overview

The pipeline metrics scraper now supports configurable pagination to handle large historical data pulls efficiently.

## Configuration Parameter

### `max_pipeline_runs`

Controls the maximum number of pipeline runs to fetch from Azure DevOps.

**Type:** `int`  
**Default:** `0` (unlimited)  
**Location:** `receivers.azuredevops.scrapers.azuredevops.max_pipeline_runs`

## Behavior

### Unlimited Mode (`max_pipeline_runs: 0`)

When set to `0`, the scraper will:
- Fetch **all available pipeline runs** within the `pipeline_lookback_days` window
- Use **automatic pagination** with 100 runs per page
- Continue fetching until no more data is available
- Ideal for **initial historical data pull** (e.g., 1 year of data)

Example:
```yaml
pipeline_metrics_enabled: true
pipeline_lookback_days: 365  # 1 year
max_pipeline_runs: 0  # Fetch all runs with pagination
```

### Limited Mode (`max_pipeline_runs: N`)

When set to a positive number, the scraper will:
- Fetch up to **N pipeline runs**
- Stop after reaching the limit
- Useful for **ongoing monitoring** after initial data load

Example:
```yaml
pipeline_metrics_enabled: true
pipeline_lookback_days: 7  # Last 7 days
max_pipeline_runs: 1000  # Limit to 1000 most recent runs
```

## Recommended Strategy

### Phase 1: Initial Historical Load
```yaml
pipeline_metrics_enabled: true
pipeline_lookback_days: 365  # 1 year of history
max_pipeline_runs: 0  # Unlimited - fetch everything
collection_interval: 3600s  # Run once per hour during initial load
```

### Phase 2: Ongoing Monitoring
After OpenSearch has the historical data, switch to:
```yaml
pipeline_metrics_enabled: true
pipeline_lookback_days: 7  # Only recent data
max_pipeline_runs: 500  # Limit to avoid re-fetching old data
collection_interval: 900s  # 15 minutes
```

## How It Works

1. **Pagination Logic:**
   - Fetches 100 runs per API request (Azure DevOps max page size)
   - Uses continuation tokens to get next page
   - Stops when: no more data, limit reached, or no continuation token

2. **Logging:**
   ```
   INFO MaxPipelineRuns is 0, will fetch all available pipeline runs with pagination
   INFO Fetched 100 build runs in this page (total: 100)
   INFO Fetched 100 build runs in this page (total: 200)
   ...
   INFO Fetched total of 1247 build runs from Builds API
   ```

3. **Performance:**
   - Each page takes ~1-2 seconds
   - 1000 runs ≈ 10-20 seconds total
   - 10,000 runs ≈ 2-3 minutes total

## OpenSearch Deduplication

OpenSearch will naturally deduplicate logs based on:
- Same `@timestamp`
- Same `attributes.job.id`
- Same `attributes.pipeline.run.id`

So re-fetching old data won't create duplicates, but it's inefficient. Use `max_pipeline_runs` to avoid this.

## Example Configurations

### Development (Quick Testing)
```yaml
pipeline_lookback_days: 1
max_pipeline_runs: 50
```

### Production (Initial Load)
```yaml
pipeline_lookback_days: 365
max_pipeline_runs: 0
```

### Production (Steady State)
```yaml
pipeline_lookback_days: 30
max_pipeline_runs: 1000
```
