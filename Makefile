# Makefile for xwebs
# Go CLI tool for WebSocket development

# Binary name
BINARY_NAME = xwebs

# Output directory
BIN_DIR = bin

# Go build variables
GO = go
GOFLAGS = -v
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

BUILD_PKG = github.com/0funct0ry/xwebs/internal/build
LDFLAGS = -X $(BUILD_PKG).Version=$(VERSION) -X $(BUILD_PKG).Commit=$(COMMIT) -X $(BUILD_PKG).BuildDate=$(BUILD_DATE)

# Directories
UI_DIR = ui
UI_DIST = ui/dist

# Default target
.PHONY: all
all: build

# Build the binary for the current platform
.PHONY: build
build:
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) .

# Build for production (with ldflags)
.PHONY: build-prod
build-prod:
	$(GO) build $(GOFLAGS) -ldflags="-s -w $(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) .

# Build for all platforms
.PHONY: build-all
build-all:
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME).exe .

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BIN_DIR)

# Install the binary to $GOPATH/bin
.PHONY: install
install:
	$(GO) install .

# Run go mod tidy
.PHONY: tidy
tidy:
	$(GO) mod tidy

# Format code
.PHONY: fmt
fmt:
	$(GO) fmt ./...

# Run go vet
.PHONY: vet
vet:
	$(GO) vet ./...

# Run linters (requires golangci-lint to be installed)
.PHONY: lint
lint:
	@which golangci-lint > /dev/null 2>&1 || (echo "golangci-lint not found, running go vet instead..." && $(GO) vet ./...)
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./... || true

# Run tests
.PHONY: test
test:
	$(GO) test -v ./...

# Run tests with coverage
.PHONY: test-cover
test-cover:
	$(GO) test -cover ./...

# Run tests with race detector
.PHONY: test-race
test-race:
	$(GO) test -race ./...

# Run benchmarks
.PHONY: bench
bench:
	$(GO) bench -bench=. ./...

# Full CI pipeline
.PHONY: ci
ci: fmt vet test build

# Development helper: run the CLI
.PHONY: run
run:
	$(GO) run .

# Development helper: run with verbose
.PHONY: run-verbose
run-verbose:
	$(GO) run . --verbose

# Help target
.PHONY: help
help:
	@echo "xwebs Makefile targets:"
	@echo "  build        - Build the binary for current platform"
	@echo "  build-prod   - Build for production (with optimizations)"
	@echo "  build-all    - Cross-compile for all platforms"
	@echo "  clean        - Remove build artifacts"
	@echo "  install      - Install to \$$GOPATH/bin"
	@echo "  tidy          - Run go mod tidy"
	@echo "  fmt           - Format code"
	@echo "  vet           - Run go vet"
	@echo "  lint          - Run linters"
	@echo "  test          - Run tests"
	@echo "  test-cover    - Run tests with coverage"
	@echo "  test-race     - Run tests with race detector"
	@echo "  bench         - Run benchmarks"
	@echo "  ci            - Full CI pipeline (fmt, vet, test, build)"
	@echo "  run           - Run the CLI"
	@echo "  help          - Show this help message"
