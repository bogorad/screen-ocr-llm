# Makefile for building screen-ocr-llm for different platforms

# Variables
BINARY_NAME=screen-ocr-llm
MAIN_PATH=./src/main

# Build for current platform
ifeq ($(OS),Windows_NT)
build:
	go build -ldflags "-H=windowsgui" -o $(BINARY_NAME).exe $(MAIN_PATH)
else
build:
	go build -o $(BINARY_NAME) $(MAIN_PATH)
endif

# Build for Windows
build-windows:
	# Build as a Windows GUI subsystem binary to hide the console window
	GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui" -o $(BINARY_NAME).exe $(MAIN_PATH)

# Build for macOS (Intel) - requires macOS or cross-compilation setup
build-macos:
	@echo "Building for macOS Intel (requires macOS or proper cross-compilation setup)..."
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -o $(BINARY_NAME)-macos-amd64 $(MAIN_PATH)

# Build for macOS (Apple Silicon) - requires macOS or cross-compilation setup
build-macos-arm:
	@echo "Building for macOS Apple Silicon (requires macOS or proper cross-compilation setup)..."
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build -o $(BINARY_NAME)-macos-arm64 $(MAIN_PATH)

# Build for Linux - requires Linux or cross-compilation setup
build-linux:
	@echo "Building for Linux (requires Linux or proper cross-compilation setup)..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o $(BINARY_NAME)-linux $(MAIN_PATH)

# Build for Linux without CGO (may have limited functionality)
build-linux-nocgo:
	@echo "Building for Linux without CGO (limited functionality)..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(BINARY_NAME)-linux-nocgo $(MAIN_PATH)

# Linux CLI build target
build-cli-linux:
	GOOS=linux GOARCH=amd64 go build -o ocr-tool ./src/cmd/cli

# Cross-compile from Windows
build-cli-linux-from-windows:
	set GOOS=linux&& set GOARCH=amd64&& go build -o ocr-tool ./src/cmd/cli

# Local build (detects OS automatically)
build-cli:
	go build -o ocr-tool ./src/cmd/cli

# Test CLI
test-cli:
	cd src/cmd/cli && go test -v

# Build for all platforms (may fail on some platforms due to CGO dependencies)
build-all: build-windows build-macos build-macos-arm build-linux

# Build for current platform only (recommended)
build-current: build

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe $(BINARY_NAME)-macos-amd64 $(BINARY_NAME)-macos-arm64 $(BINARY_NAME)-linux $(BINARY_NAME)-linux-nocgo

# Run tests
test:
	go test ./...

# Run tests with coverage
test-coverage:
	go test -cover ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Install dependencies
deps:
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Run all checks
check: fmt vet test

# Create a sample .env file
env-example:
	@if [ ! -f .env ]; then \
		echo "Creating .env from .env.example..."; \
		cp .env.example .env; \
		echo "Please edit .env with your actual API key and model."; \
	else \
		echo ".env file already exists."; \
	fi

# Default target
all: deps check build

.PHONY: build build-windows build-macos build-macos-arm build-linux build-linux-nocgo build-cli-linux build-cli-linux-from-windows build-cli test-cli build-all build-current clean test test-coverage test-verbose deps fmt vet check env-example all