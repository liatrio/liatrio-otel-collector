FROM alpine:3.18.2 as cacerts
RUN apk --update add --no-cache ca-certificates

FROM scratch

ARG UID=10001
USER ${UID}

COPY --from=cacerts /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --chmod=755 liatrio-otel-collector /usr/bin/liatrio-col
COPY config/config.yaml /etc/liatrio-otel/config.yaml


ENTRYPOINT ["/usr/bin/liatrio-col"] 
CMD ["--config=/etc/liatrio-otel/config.yaml"]
EXPOSE 4317 55678 55679
