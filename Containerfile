FROM docker.io/golang:1.23-bookworm AS builder

WORKDIR /src
COPY . /src

RUN GOBIN=/build/usr/bin go install /src/cmd/webhook-gateway

FROM docker.io/debian:bookworm-slim
RUN apt-get update -y && apt-get upgrade -y && apt-get install -y --no-install-recommends \
    ca-certificates

RUN adduser --system --group --no-create-home webhook-gateway

VOLUME /var/lib/webhook-gateway

COPY --from=builder /build /
USER webhook-gateway

ENTRYPOINT ["/usr/bin/webhook-gateway", "-config", "/var/lib/webhook-gateway/config.toml"]
