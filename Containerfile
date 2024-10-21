FROM docker.io/golang:1.23-bookworm AS builder

RUN GOBIN=/build/usr/bin go install go.deuill.org/webhook-gateway@latest
COPY gateway.conf /build/etc/webhook-gateway/config.conf

FROM docker.io/debian:bookworm-slim
RUN apt-get update -y && apt-get upgrade -y && apt-get install -y --no-install-recommends \
    ca-certificates

RUN adduser --system --group --no-create-home webhook-gateway

COPY --from=builder /build /
USER webhook-gateway

ENTRYPOINT ["/usr/bin/webhook-gateway", "-config", "/etc/webhook-gateway/config.conf"]
