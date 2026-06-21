# Project Validator Makefile

REPO_ROOT := $(abspath ../..)

CMDs := validator landscape-updater bootstrap onboarding-report audit-checker staleness-checker generate-schema migrate
BINS := $(addprefix bin/,$(CMDs))

.PHONY: all build test clean install run help provision $(CMDs)

# Default target
all: build

# Build all binaries in parallel
build:
	@mkdir -p bin
	@go build -o bin/validator ./cmd/validator & \
	go build -o bin/landscape-updater ./cmd/landscape-updater & \
	go build -o bin/bootstrap ./cmd/bootstrap & \
	go build -o bin/onboarding-report ./cmd/onboarding-report & \
	go build -o bin/audit-checker ./cmd/audit-checker & \
	go build -o bin/staleness-checker ./cmd/staleness-checker & \
	go build -o bin/generate-schema ./cmd/generate-schema & \
	go build -o bin/migrate ./cmd/migrate & \
	wait
	@echo "All binaries built."

bin/%: cmd/%/main.go $(wildcard *.go)
	@mkdir -p bin
	go build -o $@ ./cmd/$*

# Build a single tool: make validator, make bootstrap, etc.
$(CMDS): %: bin/%


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
	REPO_ROOT=$(REPO_ROOT) ./bin/validator --config testdata/projectlist.yaml --maintainers testdata/maintainers.yaml --verify-maintainers

# Run the validator and show changes only
run-changes: build
	@echo "Running validator and filtering for changes..."
	REPO_ROOT=$(REPO_ROOT) ./bin/validator --config testdata/projectlist.yaml --maintainers testdata/maintainers.yaml --verify-maintainers | grep -E "(CHANGED|INVALID|Summary)"

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
	@echo "projects:" > testdata/test-projectlist.yaml
	@echo "  - url: \"file://$(PWD)/testdata/test-project.yaml\"" >> testdata/test-projectlist.yaml
	@echo "    id: \"test-project\"" >> testdata/test-projectlist.yaml
	@echo "Test project list created: testdata/test-projectlist.yaml"

# Run validator with test setup
test-run: test-setup build
	REPO_ROOT=$(REPO_ROOT) ./bin/validator --config testdata/test-projectlist.yaml

# Provision a .project repo (prints usage if no args given)
provision: build
	@if [ -z "$(ORG)" ]; then \
		echo "Usage: make provision ORG=<org> NAME=<name> [REPO=<repo>] [DRY_RUN=1]"; \
		echo "       make provision BATCH=<file> [DRY_RUN=1]"; \
		echo ""; \
		echo "Examples:"; \
		echo "  make provision ORG=project-copacetic NAME=Copacetic DRY_RUN=1"; \
		echo "  make provision BATCH=scripts/example-batch.txt DRY_RUN=1"; \
	elif [ -n "$(BATCH)" ]; then \
		./scripts/provision.sh --batch "$(BATCH)" --bootstrap-bin bin/bootstrap $(if $(DRY_RUN),--dry-run); \
	else \
		./scripts/provision.sh --org "$(ORG)" --name "$(NAME)" $(if $(REPO),--repo "$(REPO)") --bootstrap-bin bin/bootstrap $(if $(DRY_RUN),--dry-run); \
	fi

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build all binaries in parallel"
	@echo "  <tool>        - Build a single tool (e.g., make validator, make bootstrap)"
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
	@echo "  provision     - Provision a .project repo (see: make provision)"
	@echo "  help          - Show this help message"
