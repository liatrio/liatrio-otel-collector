# githubreceiver
A custom OTEL receiver for GitHub Metrics


### Manually adding new metrics

* Go into the receiver and edit the respective `metadata.yaml` file. 
* Add your new metrics
* Run `go generate -v ./...`

> Note 1: To run `go generate -v ./...` one must have [mdatagen](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/cmd/mdatagen)
> installed. To install: 
> * clone down the [otel contrib repo](https://github.com/open-telemetry/opentelemetry-collector-contrib)
> * `cd cmd/mdatagen`
> * `go install .`

> Note 2: This is temporary while we maintain our own repository that is not full fledged. 
> Ideally the code living here would be contributed back leveraging all their actions & makefiles 
> or maintained in our own fork of the project. 
