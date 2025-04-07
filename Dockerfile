
ARG GO_VERSION=1.24.1
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS builder

# To debug DB
RUN apt-get update && apt-get install -y sqlite3

WORKDIR /app

COPY . .

RUN go mod download

WORKDIR /app/app

RUN go build -o run

CMD ["./run"]