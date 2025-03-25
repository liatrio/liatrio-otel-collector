FROM alpine:3.21.3 AS cacerts
RUN apk --update add --no-cache ca-certificates

FROM scratch

ARG BIN_PATH=liatrio-otel-collector

ARG UID=10001
USER ${UID}

COPY --from=cacerts /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --chmod=755 ${BIN_PATH} /usr/bin/liatrio-col
COPY config/config.yaml /etc/liatrio-otel/config.yaml


ENTRYPOINT ["/usr/bin/liatrio-col"] 
CMD ["--config=/etc/liatrio-otel/config.yaml"]
EXPOSE 4317 55678 55679
