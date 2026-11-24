# syntax=docker/dockerfile:1.7

# Naming (intentionally different, do not "fix"):
#   binary: landscape2-mcp-server
#   image:  ghcr.io/cncf/landscape-mcp-server

FROM golang:1.27.4-alpine AS builder

ARG VERSION

WORKDIR /app

COPY go.mod ./
COPY *.go ./

RUN LDX=""; \
    if [ -n "$VERSION" ]; then LDX="-X main.serverVersion=$VERSION"; fi; \
    CGO_ENABLED=0 go build -trimpath -ldflags "-s -w $LDX" -o landscape2-mcp-server

FROM alpine:3.24

RUN apk add --no-cache ca-certificates \
    && adduser -D -u 10001 app

WORKDIR /app

COPY --from=builder /app/landscape2-mcp-server /usr/local/bin/landscape2-mcp-server

USER 10001

ENTRYPOINT ["landscape2-mcp-server"]
