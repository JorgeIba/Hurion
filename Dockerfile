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

RUN apt-get update && apt-get install -y \
    sqlite3 \
    ca-certificates \
    cron \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /huron-bot

# Bot binary
COPY --from=builder /build/bot /huron-bot/bin/huron-bot

# Backup
COPY backup/ /huron-bot/backup/
RUN chmod +x /huron-bot/backup/*.sh

CMD ["/huron-bot/backup/entrypoint.sh"]
