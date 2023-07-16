FROM ubuntu:latest
ENTRYPOINT ["/usr/bin/otelcol"]
COPY liatrio-otel-collector /usr/bin/otelcol

