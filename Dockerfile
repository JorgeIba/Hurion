
ARG GO_VERSION=1.24.1
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS builder

WORKDIR /app
COPY ./go.mod ./
COPY ./go.sum ./

RUN go mod download


COPY ./app/main.go ./

# Compila el programa
RUN go build -o bot

CMD ["./bot"]