ARG GO_VERSION=1.24.1

# Stage 1: Build
FROM golang:${GO_VERSION} AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY app/ ./app/
WORKDIR /build/app
RUN CGO_ENABLED=1 GOOS=linux go build -o /build/bot

# Stage 2: Runtime
FROM debian:bookworm-slim

# Copy Litestream binary from official image
COPY --from=litestream/litestream:0.3.13 /usr/local/bin/litestream /usr/local/bin/litestream

RUN apt-get update && apt-get install -y \
    sqlite3 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /huron-bot

# Bot binary
COPY --from=builder /build/bot /huron-bot/bin/huron-bot

# Litestream Config & Startup Script
COPY litestream.yml /etc/litestream.yml
COPY litestream.local.yml /etc/litestream.local.yml
COPY config.yaml /huron-bot/config.yaml
COPY scripts/run.sh /huron-bot/run.sh
RUN chmod +x /huron-bot/run.sh

CMD ["/huron-bot/run.sh", "--seed"]
