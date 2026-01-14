FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /bin/validator ./cmd/validator
RUN go build -o /bin/landscape-updater ./cmd/landscape-updater

FROM alpine:3.19

WORKDIR /app

# Install git and gh cli as they are required by landscape-updater for PR creation
# ca-certificates is needed for HTTPS requests
RUN apk add --no-cache git github-cli ca-certificates

COPY --from=builder /bin/validator /usr/local/bin/validator
COPY --from=builder /bin/landscape-updater /usr/local/bin/landscape-updater

# Default entrypoint is validator, but can be overridden
ENTRYPOINT ["validator"]
