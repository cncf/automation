# syntax=docker/dockerfile:1.7

FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY *.go ./

RUN go build -o landscape2-mcp-server

FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/landscape2-mcp-server /usr/local/bin/landscape2-mcp-server

ENTRYPOINT ["landscape2-mcp-server"]
