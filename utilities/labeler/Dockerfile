# Use the official Golang image as a base image
FROM golang:1.24.5 AS builder

# Set the working directory
WORKDIR /app

# Copy the rest of the application code
COPY * /app/

# Download dependencies
RUN go mod download

# Build the Go binary
RUN go build -o labeler main.go

# Use a minimal base image for the final container
FROM gcr.io/distroless/base-debian12

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/labeler /labeler

# Set the entrypoint
ENTRYPOINT ["/labeler"]
