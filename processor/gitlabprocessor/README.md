# GitLab Pipeline Processor

## Overview

The GitLab Pipeline Processor is a custom [OpenTelemetry (OTel) Collector](https://opentelemetry.io/docs/collector/) processor designed to enrich log records with metadata about pipeline components from GitLab repositories. This enables enhanced observability and traceability for CI/CD pipeline executions by embedding component/version information directly into your logs.

## Features
- Fetches pipeline component information from GitLab for each log record containing repository and revision attributes.
- Enriches log records with component version metadata in the form:
  - `component.<component_path>.version = <version>`
- Supports all types of GitLab CI includes: **component includes**, **local includes**, and **file includes**. This means the processor extracts and annotates logs with version or source information for any component, file, or local pipeline include referenced in the `.gitlab-ci.yml` file.
- Handles missing attributes and API errors gracefully (logs errors, continues processing).

## How It Works
1. For each log record, checks for the presence of:
   - `vcs.repository.name` (the full GitLab repository path)
   - `vcs.ref.head.revision` (the commit SHA or revision)
2. If both are present, queries the GitLab GraphQL API to retrieve and parse the `.gitlab-ci.yml` pipeline definition and its includes.
3. For each discovered include (component, file, or local), adds a new attribute to the log record:
   - Example: `component.liatrio/pipeline-components/components/test.version = 1.0.0` (for a component include)
   - For file and local includes, the processor annotates the log with the appropriate version (commit SHA or 'local') and path information.
4. If required attributes are missing, or an error occurs, the processor logs the error and continues.

## Example Use Case
Organizations using GitLab pipelines can use this processor to automatically annotate pipeline logs with precise component and version information, enabling better debugging, auditing, and compliance tracking for CI/CD workflows.

## Configuration Example
Below is an example of how to use the GitLab Pipeline Processor in your OTel Collector configuration:

```yaml
receivers:
  otlp:
    protocols:
      grpc:

processors:
  gitlabpipeline:
    token: ${env:GL_PAT}
    # Alternatively, set the token directly:
    # token: "glpat-xxxxxxxxxxxxxxxxxxxx"

exporters:
  logging:
    loglevel: debug

service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [gitlabpipeline]
      exporters: [logging]
```

## Requirements
- OpenTelemetry Collector
- Access to the GitLab GraphQL API (ensure the token provided has appropriate permissions)

## Development & Testing
- To build the processor:
  ```sh
  go build ./...
  ```
- To run tests:
  ```sh
  go test ./...
  ```

## License
This processor is distributed under the Apache 2.0 License.

## Contact
For questions or support, please contact the Liatrio engineering team.
