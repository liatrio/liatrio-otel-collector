---

<p align="center">
  <a href="https://github.com/liatrio/liatrio-otel-collector/actions/workflows/build.yml?query=branch%3Amain">
    <img alt="Build Status" src="https://img.shields.io/github/actions/workflow/status/liatrio/liatrio-otel-collector/build.yml?branch=main&style=for-the-badge">
  </a>
  <a href="https://goreportcard.com/report/github.com/liatrio/liatrio-otel-collector/pkg/receiver/gitproviderreceiver">
    <img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/liatrio/liatrio-otel-collector/pkg/receiver/gitproviderreceiver?style=for-the-badge">
  </a>
  <a href="https://github.com/liatrio/liatrio-otel-collector/releases">
    <img alt="GitHub release" src="https://img.shields.io/github/v/release/liatrio/liatrio-otel-collector?include_prereleases&style=for-the-badge">
  </a>
  <a href="https://api.securityscorecards.dev/projects/github.com/liatrio/liatrio-otel-collector/badge">
    <img alt="OpenSSF Scorecard" src="https://img.shields.io/ossf-scorecard/github.com/liatrio/liatrio-otel-collector?label=openssf%20scorecard&style=for-the-badge">
  </a>
</p>

---

# liatrio-otel-collector

The Liatrio OTEL Collector is an upstream distribution of the Open Telemetry collector, rebuilt with custom packages hosted within this repository.  These custom packages are by default targeted for downstream contribution to Open Telemetry; pursuant acceptance by the community.

## Quick Start Guide

Before diving into the codebase, you'll need to set up a few things. This guide is designed to help you get up and running in no time!

Here are the steps to quickly get started with this project:

1. **Install Go:** Use Homebrew to install Go with the following command:

    ```bash
    brew install go
    ```

2. **Install pre-commit:** With the help of Homebrew, install the pre-commit as follows:

    ```bash
    brew install pre-commit
    ```

3. **Initialize pre-commit:** Once installed, initiate pre-commit by running:

    ```bash
    pre-commit install
    ```

4. Set up [GitHub][0] and/or [GitLab][1] scraper(s).

5. **Start the collector:** Finally, initiate the program:

    ```bash
    make run
    ```

### Configure GitHub Scraper

1. **Generate a Personal Access Token (PAT) for GitHub:** Ensure it only has read access.

2. **Set your GitHub username and PAT as environment variables:** Replace `<user>` with your GitHub username and `<pat>` with your generated PAT, then run the following commands:

    ```bash
    export GH_USER=<user>
    export GH_PAT=<pat>
    ```

### Configure GitLab Scraper

1. **Generate a Personal Access Token (PAT) for GitLab**

2. **Set your GitLab pat as an environment variable:**

   ```bash
   export GITLAB_PAT=<pat>
   ```

> **Note:** If you wish to terminate the program, you can use the `ctrl+c` command in the terminal. Keep in mind that it might take a few moments for the program to exit gracefully.

### Exporting to Grafana Cloud

If you want to export your data to Grafana Cloud through their OTLP endpoint, there's a couple of extra things you'll need to do.

1. Run `export GRAF_USER` and `export GRAF_PAT` with your instance id and cloud api key
2. Update the [config/config.yaml][2] file with the following:

```yaml
extensions:
  ...

  basicauth/grafana:
    client_auth:
      username: ${env:GRAF_USER}
      password: ${env:GRAF_PAT}
...

exporters:
  ...
  otlphttp:
    auth:
      authenticator: basicauth/grafana
    endpoint: https://otlp-gateway-prod-us-central-0.grafana.net/otlp

...

service:
  extensions: [..., basicauth/grafana]
  pipelines:
    metrics:
      receivers: [otlp, gitprovider]
      processors: []
      exporters: [..., otlphttp]

```

### Debugging

To debug through `vscode`:

* run `make build-debug`
* run `cd build && code .`
* run vscode debugger for go

## OTEL Intro

OTEL is a protocol used for distributed logging, tracing, and metrics.
To collect metrics from various services, we need to configure receivers.
OTEL provides many built-in receivers, but in certain cases, we may need to
create custom receivers to meet our specific requirements.

A custom receiver in OTEL is a program that listens to a specific endpoint and
receives incoming log, metrics, or trace data. This receiver then pipes the
content to a process, for it to then be exported to a data backend.

Creating a custom receiver in OTEL involves implementing the receiver interface
and defining the endpoint where the receiver will listen for incoming trace data.
Once the receiver is implemented, it can be deployed to a specific location and
configured to start receiving trace data.

### Prereqs

There is currently a guide to build a custom trace receiver. It is a long read,
requires a fairly deep understanding, and is slightly out of date due to
non-backwards compatible internal API breaking changes. This document and
receiver example attempts to simplify that to an extent.

There are a few main concepts that should help you get started:

1. Get familiar with the `ocb` tool. It is used to build custom collectors using a `build-config.yaml` file.
2. Get familiar with `Go` & the `Factory` design pattern.
3. Clearly define what outcome you want before building a customization.
4. Get familiar with `Go interfaces`.
5. Get familiar with `delv` the go debugger.

### References & Useful Resources

* [builder command - go.opentelemetry.io/collector/cmd/builder - Go Packages][3]
* [Building a Trace Receiver][4]
* [Building a custom collector][5]
* [otel4devs/builder-config.yaml at main · rquedas/otel4devs][6]
* [opentelemetry-collector-contrib/receiver/activedirectorydsreceiver at main · open-telemetry/opentelemetry-collector-contrib][7]
* [opentelemetry-collector-contrib/extension/basicauthextension at main · open-telemetry/opentelemetry-collector-contrib][8]

[0]: #configure-github-scraper
[1]: #configure-gitlab-scraper
[2]: ./config/config.yaml
[3]: https://pkg.go.dev/go.opentelemetry.io/collector/cmd/builder#section-readme
[4]: https://opentelemetry.io/docs/collector/trace-receiver/#representing-operations-with-spans
[5]: https://opentelemetry.io/docs/collector/custom-collector/
[6]: https://github.com/rquedas/otel4devs/blob/main/collector/receiver/trace-receiver/builder-config.yaml
[7]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/activedirectorydsreceiver
[8]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/extension/basicauthextension
