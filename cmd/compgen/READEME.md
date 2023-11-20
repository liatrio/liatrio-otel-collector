# Compgen

Compgen is a tool for building new otel components.

## Usage

    compgen receiver [receiver-name]

## Adding Commands

Compgen is built on the [Cobra](https://github.com/spf13/cobra) library. While new commands can be added manually, it may be easier to use the [cobra-cli](https://github.com/spf13/cobra-cli/blob/main/README.md). New commands can be added to compgen by running the following shell commands:

    cd cmd/compgen
    cobra-cli add [command-name]
    cobra-cli add [command-name] -p [parent-command-name]
