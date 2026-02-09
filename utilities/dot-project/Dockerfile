# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy dependency files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binaries
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/validator ./cmd/validator
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/landscape-updater ./cmd/landscape-updater

# Runtime stage
FROM alpine:3.20

WORKDIR /app

# Install git and gh cli as they are required by landscape-updater for PR creation
# ca-certificates is needed for HTTPS requests
RUN apk add --no-cache git github-cli ca-certificates

COPY --from=builder /bin/validator /usr/local/bin/validator
COPY --from=builder /bin/landscape-updater /usr/local/bin/landscape-updater

# Default entrypoint is validator, but can be overridden
ENTRYPOINT ["validator"]
