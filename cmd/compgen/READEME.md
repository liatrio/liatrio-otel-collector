# Compgen

Compgen is a tool for building new otel components.

## Usage

    compgen receiver [receiver-name]

The `receiver-name` should be a full [module path](https://go.dev/ref/mod#glos-module-path). For example, if you wish to build a new receiver with a short name of `SomeNewReceiver`, then pass this string to the receiver-name argument: `github.com/liatrio/liatrio-otel-collector/pkg/receiver/SomeNewReceiver`

## Adding Commands

Compgen is built on the [Cobra](https://github.com/spf13/cobra) library. While new commands can be added manually, it may be easier to use the [cobra-cli](https://github.com/spf13/cobra-cli/blob/main/README.md) instead. New commands can be added to compgen by running the following shell commands:

    cd cmd/compgen
    cobra-cli add [command-name]
    cobra-cli add [command-name] -p [parent-command-name]
