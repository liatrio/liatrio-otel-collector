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

```sh
Compgen is a tool for building new receivers, processors, and exporters for Open Telemetry.

Usage:
  compgen [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  receiver    Build a new Open Telemetry receiver component

Flags:
  -h, --help   help for compgen

Use "compgen [command] --help" for more information about a command.
```

## Naming Conventions for Open Telemetry Components

New component names are passed to compgen via command line arguments. These are expected to be a full [module paths](https://go.dev/ref/mod#glos-module-path) from which compgen will extract a short name to use in code generation.

For example, if you wish to build a new receiver with a short name of `myreceiver`, then supply this string to the receiver subcommand: `github.com/liatrio/liatrio-otel-collector/receiver/myreceiver`

## After Running Compgen

### Makefile

Components are expected to pass a series of tests defined by Makefile.Common. Your component's Makefile must import Makefile.common. An example is provided as a commment.

### Metadata.yaml

Compgen's includes templates for metadata.yaml. This file is used by [mdatagen](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/cmd/mdatagen) to generate aditional code for the component. See the [README](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/cmd/mdatagen/README.md) for details on how to use mdatagen.

You can run mdatagen by running `make gen`.

### Build Component Logic

Compgen's component template are very limited by design. They are intended to supply the minimum amount of code required to compile, start, and run the component in a collector. Component developers are required to add all functionality to fulfill the component's purpose. See [OpenTelemetry's Building Custom Components](https://opentelemetry.io/docs/collector/building/) page for detailed information and examples of how to build new components.

### Building Into The OTEL Binary

When ready, you can compile your new component into Open Telemetry using OCB. Update `config/manifest.yaml` with configurations appropriate for your component. For example, a new receiver named `myreceiver` may require this added yaml:

```yaml
receivers:
  - gomod: github.com/liatrio/liatrio-otel-collector/receiver/myreceiver v0.1.0
```

This will instruct OCB to include `myreceiver` when compiling Open Telemetry. Continuing the example, include the following configuration to instruct OCB to use the local code:

```yaml
replaces:
  - github.com/liatrio/liatrio-otel-collector/receiver/myreceiver => ../receiver/myreceiver/
```

See [OpenTelemetry's Building a Custom Collector](https://opentelemetry.io/docs/collector/custom-collector/) documentation for additional guidance.

## Contributing to Compgen

### Adding Commands to Compgen

Compgen is built on the [Cobra](https://github.com/spf13/cobra) library. While new commands can be added manually, it may be easier to use the [cobra-cli](https://github.com/spf13/cobra-cli/blob/main/README.md) instead. New commands can be added to compgen by running the following shell commands:

```sh
cd cmd/compgen
cobra-cli add [command-name]
cobra-cli add [command-name] -p [parent-command-name]
```

### Adding Templates to Compgen

New compgen commands are expected to be paired with new templates for Open Telemetry compnents. These templates should include the minimum functionality required to compile, start, and run the component in a collector. Conversely, these templates should include the maximum supporting code expected for the component, such as README and Makefile templates.
