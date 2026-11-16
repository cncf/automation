BINARY := landscape2-mcp-server

.PHONY: build test vet fmt-check lint ci

build:
	go build -o $(BINARY)

test:
	go test -race -count=1 ./...

vet:
	go vet ./...

fmt-check:
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run; else echo "golangci-lint not installed, skipping"; fi

ci: fmt-check vet lint test
