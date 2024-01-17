module github.com/liatrio/liatrio-otel-collector

go 1.21

toolchain go1.21.5

require github.com/liatrio/liatrio-otel-collector/receiver/gitproviderreceiver v0.38.1

replace github.com/liatrio/liatrio-otel-collector/receiver/gitproviderreceiver => ./receiver/gitproviderreceiver
