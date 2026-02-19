# Local Development Environment Setup

This guide explains how to set up a local development environment for the Liatrio OpenTelemetry Collector with OpenSearch, Grafana, and Prometheus for testing Azure DevOps pipeline metrics.

## Prerequisites

- Docker and Docker Compose installed
- Go 1.21+ (for building the collector)
- Make (for running build commands)
- Azure DevOps Personal Access Token (PAT) with appropriate permissions

## Quick Start

### 1. Set Environment Variables

Create a `.env` file in the project root or export these variables:

```bash
export ADO_PAT="your-azure-devops-pat"
export ADO_ORG="your-organization"
export ADO_PROJECT="your-project"
```

**Required Azure DevOps Permissions:**
- Code: Read
- Build: Read
- Release: Read (if using release pipelines)

### 2. Start the Infrastructure Stack

Start OpenSearch, Grafana, and Prometheus:

```bash
docker-compose up -d
```

This will start:
- **OpenSearch** on ports 9200 (API) and 9600 (Performance Analyzer)
- **OpenSearch Dashboards** on port 5601
- **Grafana** on port 3000
- **Prometheus** on port 9090

Wait for all services to be healthy (check with `docker-compose ps`).

### 3. Build and Run the Collector

Build the collector:

```bash
make build
```

Run the collector with the local development configuration:

```bash
./build/otelcol-custom --config=config/config-local-dev.yaml
```

**Note**: The `make run` command always uses `config/config.yaml`. To use a different config, run the binary directly as shown above.

## Accessing Services

### Grafana
- **URL**: http://localhost:3000
- **Username**: `admin`
- **Password**: `admin`
- **Features**: Pre-configured datasources for OpenSearch and Prometheus

### OpenSearch
- **API URL**: http://localhost:9200
- **Dashboards URL**: http://localhost:5601
- **Security**: Disabled for local development

### Prometheus
- **URL**: http://localhost:9090
- **Targets**: Configured to scrape OTEL collector metrics

### OTEL Collector
- **Metrics Endpoint**: http://localhost:8888/metrics
- **OTLP gRPC**: localhost:4317
- **OTLP HTTP**: localhost:4318
- **Prometheus Exporter**: localhost:8889

## Verifying the Setup

### 1. Check Service Health

```bash
# Check all services are running
docker-compose ps

# Check OpenSearch health
curl http://localhost:9200/_cluster/health

# Check Prometheus targets
curl http://localhost:9090/api/v1/targets
```

### 2. Verify Logs are Flowing to OpenSearch

Once the collector is running and scraping data:

```bash
# Check if logs index exists
curl http://localhost:9200/_cat/indices?v

# Search for pipeline logs
curl -X GET "http://localhost:9200/otel-logs/_search?pretty" -H 'Content-Type: application/json' -d'
{
  "query": {
    "match_all": {}
  },
  "size": 10
}'

# Count logs
curl http://localhost:9200/otel-logs/_count
```

### 3. Verify Metrics in Prometheus

1. Open http://localhost:9090
2. Go to Status → Targets
3. Verify `otel-collector` target is UP
4. Query for metrics: `otel_vcs_repository_count`

### 4. Verify Grafana Datasources

1. Open http://localhost:3000
2. Go to Configuration → Data Sources
3. Verify both OpenSearch and Prometheus datasources show "Data source is working"

## Development Workflow

### Making Changes to the Collector

1. Make code changes
2. Rebuild: `make build`
3. Restart the collector (Ctrl+C and rerun)
4. Verify changes in Grafana or via API queries

### Viewing Collector Logs

The collector runs with debug logging enabled. Watch for:
- Scraper initialization messages
- API call logs
- Error messages
- Data export confirmations

### Iterating on Configuration

1. Edit `config/config-local-dev.yaml`
2. Restart the collector
3. Verify configuration changes took effect

## Troubleshooting

### Services Won't Start

```bash
# Check logs for specific service
docker-compose logs opensearch
docker-compose logs grafana
docker-compose logs prometheus

# Restart all services
docker-compose down
docker-compose up -d
```

### OpenSearch Connection Refused

- Wait 30-60 seconds after `docker-compose up` for OpenSearch to initialize
- Check health: `curl http://localhost:9200/_cluster/health`
- Check logs: `docker-compose logs opensearch`

### Collector Can't Connect to OpenSearch

- Verify OpenSearch is running: `docker-compose ps`
- Check network connectivity: `docker network inspect liatrio-otel-collector_otel-network`
- Ensure collector is using correct endpoint: `http://opensearch:9200` (from Docker) or `http://localhost:9200` (from host)

### No Data in Grafana

1. Verify collector is running and scraping
2. Check collector logs for errors
3. Verify data exists in OpenSearch: `curl http://localhost:9200/otel-logs/_count`
4. Check Grafana datasource connection
5. Verify query syntax in Grafana panels

### Azure DevOps API Rate Limiting

If you see rate limit errors:
- Increase `collection_interval` in config
- Reduce the number of repositories being scraped
- Check Azure DevOps API rate limits for your organization

## Stopping the Environment

```bash
# Stop the collector (Ctrl+C in the terminal where it's running)

# Stop Docker services
docker-compose down

# Stop and remove volumes (WARNING: deletes all data)
docker-compose down -v
```

## Configuration Files

- **Docker Compose**: `docker-compose.yml`
- **OTEL Collector**: `config/config-local-dev.yaml`
- **Prometheus**: `prometheus/prometheus.yml`
- **Grafana Datasources**: `grafana/provisioning/datasources/`
- **Grafana Dashboards**: `grafana/dashboards/`

## Next Steps

- Configure pipeline metrics in Task 4.0
- Create Grafana dashboards in Task 5.0
- Add custom queries and visualizations
- Export dashboards for sharing

## Additional Resources

- [OpenTelemetry Collector Documentation](https://opentelemetry.io/docs/collector/)
- [OpenSearch Documentation](https://opensearch.org/docs/latest/)
- [Grafana Documentation](https://grafana.com/docs/)
- [Azure DevOps REST API](https://learn.microsoft.com/en-us/rest/api/azure/devops/)
