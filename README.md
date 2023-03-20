# ldapreceiver
A custom OTEL receiver for LDAP search metrics.

## Intro
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
3. Clearly define what you want to do
4. Get familiar with `Go interfaces`
5. Get familiar with `delv` the go debugger

Let’s begin

## Getting Started

To create a custom receiver, you’ll need to create a basic file structure that mirrors the following:

```markdown

├── README.md
├── config.go #where your receiver config Types will go alongside any config validations
├── config_test.go #the test file for your configs
├── factory.go #the file where your factory will go, this will implement the factory interface, configs, and component interface
├── factory_test.go #the file used to test your factory
├── receiver.go #the core code for the receiver component you're writing
├── receiver_test.go #test for the receiver
├── go.mod
└── go.sum
```

To get the `go.mod` & `go.sum` you’d follow the normal process of running 
`go mod init <my module>` . The command I ran was 
`go mod init github.com/liatrio/ldapreceive](http://github.com/liatrio/ldapreceiver`
for this custom receiver.

To be able to run & test the collector locally, you’ll want to leverage the `ocb` 
utility. You can go follow 
[https://opentelemetry.io/docs/collector/custom-collector/](https://opentelemetry.io/docs/collector/custom-collector/) 
or run the following commands:

```bash
curl -L https://github.com/open-telemetry/opentelemetry-collector/releases/download/cmd%2Fbuilder%2Fv0.73.0/ocb_0.73.0_darwin_arm64 \
-o ocb \
&& chmod +x ocb \
&& ocb --help
```

This command will let you build a custom collector with your custom receiver, 
processor, exporter or all of the above. It’s a bit finicky though, and a key 
is getting the `build-config.yaml` right. Below is an example file that I used for the ldapreceiver. 

```bash
dist:
  name: dev-otelcol
  description: Basic OTel Collector distribution for Developers
  output_path: ./dev-otelcol

    
exporters:
  - gomod: go.opentelemetry.io/collector/exporter/loggingexporter v0.73.0

processors:
  - gomod: go.opentelemetry.io/collector/processor/batchprocessor v0.73.0

receivers:
  - gomod: go.opentelemetry.io/collector/receiver/otlpreceiver v0.73.0
  - gomod: github.com/liatrio/ldapreceiver v0.1.0

replaces:
  - github.com/liatrio/ldapreceiver => <path to my local copy of ldapreceiver/ldapreceiver/ >
```

The key here is the replace statement for local development. 
Also, you need to have a version otherwise the builder & go will complain. 
The replace statement lets you update local code for your receiver in development. 

Simply run the following to create your collector:

```bash
./ocb --config builder-config.yaml
```

Inside of the newly created `dev-otelcol` directory, you’ll find the following files:

```bash

├── components.go
├── components_test.go
├── dev-otelcol
├── go.mod
├── go.sum
├── main.go
├── main_others.go
└── main_windows.go

```

You’ll want to make sure you add a `config.yaml` file in this directory which 
should look similar to the what’s defined below. This file is a basic 
configuration file for an OTEL collector, but with your specific settings. 

```bash
receivers:
  otlp:
    protocols:
      grpc:
  ldap:
    interval: 10s
    
processors:
  
exporters:
  logging:
    verbosity: detailed

service:
  pipelines:
    metrics:
      receivers: [otlp, ldap]
      processors: []
      exporters: [logging]

  telemetry:
    logs:
      level: debug
```

You can tweak it however you want, but note that if your receiver doesn’t work, 
you’ll probably get a panic. That’s why I left `otlp` in there as a receiver 
for early testing before being able to transfer to using the ldap receiver. 
There were some original weird issues just getting the thing to import. 

To get your collector running run either of the following commands:

```bash
task run #leveraging the Taskfile.yml run statement
# or
cd dev-otelcol && ./dev-otelcol --config config.yaml
```

Now we get to write some code, let the games begin. 

### Building out the receiver

**Left off here:** 

[https://opentelemetry.io/docs/collector/trace-receiver/#keeping-information-passed-by-the-receivers-factory](https://opentelemetry.io/docs/collector/trace-receiver/#keeping-information-passed-by-the-receivers-factory)

### Debugging

Debugging is essential, and you can debug your collector with custom components locally. 

Easiest way is to run the `task build-debug` and then `cd dev-otelcol` 

**IMPORTANT** the `build-config.yaml` must have the `debug_compilation: true` set to have the collector built properly. That also presumes that you’re attaching to what you compiled. VScode can build it on the fly when running debugging.

Place a `.vscode` folder into the root of that directory, with a `launch.json` file containing the following contents. (you can adjust accordingly)

```bash
{
    // Use IntelliSense to learn about possible attributes.
    // Hover to view descriptions of existing attributes.
    // For more information, visit: https://go.microsoft.com/fwlink/?linkid=830387
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch dev collector",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${fileDirname}",
            "args": ["--config", "../testdata/config.yaml"]
        }
    ]
}
```

From there you can open one of the files generated in the collector folder & run debug.

### LDAP Testing Locally (for ldapreceiver)

Used this container image: 

[GitHub - osixia/docker-openldap: OpenLDAP container image 🐳🌴](https://github.com/osixia/docker-openldap#quick-start)

Left off at being able to connect via LDAPS through the go code, and need to populate with a couple groups & users to test the search functionality. 

- [ ]  also need to write the ldap unit test
- [ ]  Basic authenticator should be used

### References & Useful Resources

[builder command - go.opentelemetry.io/collector/cmd/builder - Go Packages](https://pkg.go.dev/go.opentelemetry.io/collector/cmd/builder#section-readme)

[Building a Trace Receiver](https://opentelemetry.io/docs/collector/trace-receiver/#representing-operations-with-spans)

[Building a custom collector](https://opentelemetry.io/docs/collector/custom-collector/)

[otel4devs/builder-config.yaml at main · rquedas/otel4devs](https://github.com/rquedas/otel4devs/blob/main/collector/receiver/trace-receiver/builder-config.yaml)

[opentelemetry-collector-contrib/receiver/activedirectorydsreceiver at main · open-telemetry/opentelemetry-collector-contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/activedirectorydsreceiver)

[opentelemetry-collector-contrib/extension/basicauthextension at main · open-telemetry/opentelemetry-collector-contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/extension/basicauthextension)
