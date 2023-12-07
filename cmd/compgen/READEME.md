# Compgen

Compgen is a tool for building new otel components. The following component types are currently supported:

- [Receivers](https://opentelemetry.io/docs/collector/configuration/#receivers)
  - Pull (Scraper)
    - [ ] Logs
    - [x] Metrics
    - [ ] Traces
  - Push
    - [ ] Logs
    - [ ] Metrics
    - [ ] Traces
- [ ] [Processors](https://opentelemetry.io/docs/collector/configuration/#processors)
- [ ] [Exporters](https://opentelemetry.io/docs/collector/configuration/#exporters)

## Usage

    Compgen is a tool for building new receivers, processors, and 
        exporters for Open Telemetry.

    Usage:
      compgen [command]

    Available Commands:
      completion  Generate the autocompletion script for the specified shell
      help        Help about any command
      receiver    Build a new Open Telemetry receiver component

    Flags:
      -h, --help   help for compgen

    Use "compgen [command] --help" for more information about a command.

## Adding Commands

Compgen is built on the [Cobra](https://github.com/spf13/cobra) library. While new commands can be added manually, it may be easier to use the [cobra-cli](https://github.com/spf13/cobra-cli/blob/main/README.md) instead. New commands can be added to compgen by running the following shell commands:

    cd cmd/compgen
    cobra-cli add [command-name]
    cobra-cli add [command-name] -p [parent-command-name]
