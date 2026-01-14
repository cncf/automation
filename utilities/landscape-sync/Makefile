.PHONY: build run test clean report yaml json

BINARY_NAME=landscape-sync
OUTPUT_DIR=bin

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(OUTPUT_DIR)
	go build -o $(OUTPUT_DIR)/$(BINARY_NAME) .

run: build
	@echo "Running landscape sync check..."
	./$(OUTPUT_DIR)/$(BINARY_NAME) --verbose

report: build
	@echo "Generating missing projects report..."
	./$(OUTPUT_DIR)/$(BINARY_NAME) --output=missing-projects.md

yaml: build
	@echo "Generating YAML entries for missing projects..."
	./$(OUTPUT_DIR)/$(BINARY_NAME) --yaml --output=missing-entries.yaml

json: build
	@echo "Generating JSON output..."
	./$(OUTPUT_DIR)/$(BINARY_NAME) --json --output=missing-projects.json

test:
	go test -v ./...

clean:
	rm -rf $(OUTPUT_DIR)
	rm -f missing-projects.md missing-entries.yaml missing-projects.json

deps:
	go mod download
	go mod tidy
