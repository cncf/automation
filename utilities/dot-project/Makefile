# Project Validator Makefile

REPO_ROOT := $(abspath ../..)

.PHONY: all build test clean install run help

# Default target
all: build

# Build the validator
build:
	@echo "Building project validator..."
	@mkdir -p bin
	go build -o bin/validator ./cmd/validator
	@echo "Building landscape updater..."
	go build -o bin/landscape-updater ./cmd/landscape-updater
	@echo "Building bootstrap tool..."
	go build -o bin/bootstrap ./cmd/bootstrap

# Build docker image
docker-build:
	@echo "Building docker image..."
	docker build -t dot-project-validator .

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f validator
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -rf .cache
	rm -rf .test-cache

# Install dependencies
install:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Run the validator with default settings
run: build
	@echo "Running validator..."
	REPO_ROOT=$(REPO_ROOT) ./bin/validator --config yaml/projectlist.yaml --maintainers yaml/maintainers.yaml --verify-maintainers

# Run the validator and show changes only
run-changes: build
	@echo "Running validator and filtering for changes..."
	REPO_ROOT=$(REPO_ROOT) ./bin/validator --config yaml/projectlist.yaml --maintainers yaml/maintainers.yaml --verify-maintainers | grep -E "(CHANGED|INVALID|Summary)"

# Validate and format Go code
fmt:
	@echo "Formatting Go code..."
	go fmt ./...

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

# Check for security issues
security:
	@echo "Running security checks..."
	gosec ./...

# Create a test project list with local files
test-setup:
	@echo "Setting up test environment..."
	@echo "projects:" > yaml/test-projectlist.yaml
	@echo "  - url: \"file://$(PWD)/yaml/test-project.yaml\"" >> yaml/test-projectlist.yaml
	@echo "    id: \"test-project\"" >> yaml/test-projectlist.yaml
	@echo "Test project list created: yaml/test-projectlist.yaml"

# Run validator with test setup
test-run: test-setup build
	REPO_ROOT=$(REPO_ROOT) ./bin/validator --config yaml/test-projectlist.yaml

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the validator, landscape-updater, and bootstrap binaries"
	@echo "  docker-build  - Build docker image"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  clean         - Clean build artifacts"
	@echo "  install       - Install dependencies"
	@echo "  run           - Run validator with default settings"
	@echo "  run-changes   - Show only changes and summary"
	@echo "  fmt           - Format Go code"
	@echo "  lint          - Run linter"
	@echo "  security      - Run security checks"
	@echo "  test-setup    - Create test environment"
	@echo "  test-run      - Run with test setup"
	@echo "  help          - Show this help message"
