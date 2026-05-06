BINARY_NAME=release-checker
VERSION?=1.0.0
BUILD_DIR=build
DIST_DIR=dist
REP_DIR=reports

.PHONY: all build clean test install package help

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

install: build
	@echo "Installing to /usr/local/bin..."
	@install $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "Installed successfully"

test:
	@echo "Running tests..."
	@mkdir -p $(REP_DIR)
	@go test -v -coverprofile=$(REP_DIR)/coverage.out ./...
	@go tool cover -html=$(REP_DIR)/coverage.out -o $(REP_DIR)/coverage.html
	@echo "Coverage report generated: $(REP_DIR)/coverage.html"

test-short:
	@echo "Running short tests..."
	@go test -short ./...

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR) $(REP_DIR)
	@go clean
	@echo "Cleaned"

package: build
	@echo "Creating distribution packages..."
	@mkdir -p $(DIST_DIR)
	@tar -czf $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)
	@echo "Package created: $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz"

deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

fmt:
	@echo "Formatting code..."
	@go fmt ./...

lint:
	@echo "Running linter..."
	@go vet ./...
	@if command -v staticcheck >/dev/null; then staticcheck ./...; fi

help:
	@echo "Available targets:"
	@echo "  build       - Build the binary"
	@echo "  install     - Build and install to /usr/local/bin"
	@echo "  test        - Run tests with coverage"
	@echo "  test-short  - Run short tests (skip integration)"
	@echo "  clean       - Remove build artifacts"
	@echo "  package     - Create distribution tarball"
	@echo "  deps        - Download and tidy dependencies"
	@echo "  fmt         - Format source code"
	@echo "  lint        - Run go vet and staticcheck"
	@echo "  help        - Show this help message"
