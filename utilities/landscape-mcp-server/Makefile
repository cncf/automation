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
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint is required: https://golangci-lint.run/welcome/install/"; exit 1; }
	golangci-lint run

ci: fmt-check vet lint test
