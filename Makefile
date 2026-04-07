# Vaulty Makefile

# Variables
BINARY_NAME=vty
VERSION=0.3.0
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILT=$(shell date +%Y-%m-%d)
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(BUILT)"
CGO_ENABLED=0
BUILD_DIR=bin

# Default target
.DEFAULT_GOAL:=build

# Build for current platform
.PHONY: build
build:
	@echo "Building $(BINARY_NAME) for current platform..."
	CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/vty

# Build for all platforms
.PHONY: build-all
build-all: build-darwin-amd64 build-darwin-arm64 build-linux-amd64 build-linux-arm64 build-windows-amd64

.PHONY: build-darwin-amd64
build-darwin-amd64:
	@echo "Building $(BINARY_NAME) for darwin/amd64..."
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/vty

.PHONY: build-darwin-arm64
build-darwin-arm64:
	@echo "Building $(BINARY_NAME) for darwin/arm64..."
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/vty

.PHONY: build-linux-amd64
build-linux-amd64:
	@echo "Building $(BINARY_NAME) for linux/amd64..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/vty

.PHONY: build-linux-arm64
build-linux-arm64:
	@echo "Building $(BINARY_NAME) for linux/arm64..."
	GOOS=linux GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/vty

.PHONY: build-windows-amd64
build-windows-amd64:
	@echo "Building $(BINARY_NAME) for windows/amd64..."
	GOOS=windows GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/vty

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)

# Install binary
.PHONY: install
install:
	@echo "Installing $(BINARY_NAME)..."
	CGO_ENABLED=$(CGO_ENABLED) go install $(LDFLAGS) ./cmd/vty

# Help target
.PHONY: help
help:
	@echo "Vaulty Makefile targets:"
	@echo "  build       - Build for current platform"
	@echo "  build-all   - Build for all platforms (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64)"
	@echo "  test        - Run go tests"
	@echo "  clean       - Remove build artifacts"
	@echo "  install     - Install binary using go install"
	@echo "  help        - Show this help message"
