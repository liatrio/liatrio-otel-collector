# GitLab Pipeline Processor

## Overview

The GitLab Pipeline Processor is a custom [OpenTelemetry (OTel) Collector](https://opentelemetry.io/docs/collector/) processor designed to enrich log records with additional pipeline metadata not included in native gitlab pipeline event logs. 

## Features
- Fetches pipeline [include](https://docs.gitlab.com/ci/yaml/includes) information from GitLab for any log record containing repository and revision attributes.
- Enriches log records with include name and version metadata
  - Example (component include): `component.<component_path>.version = <version>`
- Other supported include types: 
  - **local includes**,
  - **file includes**. 

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