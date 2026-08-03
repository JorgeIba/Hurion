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
    make \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /hurion

# Bot binary & Makefile
COPY --from=builder /build/bot /hurion/bin/hurion
COPY Makefile /hurion/Makefile

# Litestream Config & Startup Script
COPY litestream.yml /etc/litestream.yml
COPY litestream.local.yml /etc/litestream.local.yml
COPY scripts/run.sh /hurion/run.sh
RUN chmod +x /hurion/run.sh

ENTRYPOINT ["/hurion/run.sh"]
