# Build Instructions

This document describes how to build the Screen OCR LLM tool from source.

## Prerequisites

- Go 1.21 or later
- Make (optional, but recommended)
- Git

## Quick Start

1. Clone the repository:
   ```bash
   git clone https://github.com/cherjr/screen-ocr-llm.git
   cd screen-ocr-llm
   ```

2. Install dependencies:
   ```bash
   make deps
   # or manually: go mod tidy
   ```

3. Build for current platform:
   ```bash
   make build
   # or manually: go build -o screen-ocr-llm main/main.go
   ```

## Available Make Targets

### Building
- `make build` - Build for current platform
- `make build-windows` - Build for Windows (produces .exe)
- `make build-macos` - Build for macOS Intel
- `make build-macos-arm` - Build for macOS Apple Silicon
- `make build-linux` - Build for Linux
- `make build-all` - Build for all platforms (may fail due to CGO dependencies)

### Testing
- `make test` - Run all tests
- `make test-verbose` - Run tests with verbose output
- `make test-coverage` - Run tests with coverage report

### Code Quality
- `make fmt` - Format code
- `make vet` - Run go vet
- `make check` - Run fmt, vet, and test

### Utilities
- `make clean` - Remove build artifacts
- `make deps` - Install/update dependencies
- `make env-example` - Create .env file from .env.example

## Cross-Platform Compilation

Due to CGO dependencies (required for GUI, clipboard, and hotkey functionality), cross-compilation requires proper toolchains for each target platform:

- **Windows**: Can be built on Windows natively
- **macOS**: Requires macOS or proper cross-compilation setup
- **Linux**: Requires Linux or proper cross-compilation setup

For development and testing, build on your current platform using `make build`.

## Configuration

Before running the built executable, you need to configure it:

1. Create a `.env` file:
   ```bash
   make env-example
   ```

2. Edit `.env` with your OpenRouter API key and preferred model:
   ```
   OPENROUTER_API_KEY=your_api_key_here
   MODEL=qwen/qwen2.5-vl-72b-instruct:free
   ```

## Running

After building and configuring:

```bash
./screen-ocr-llm        # Linux/macOS
screen-ocr-llm.exe      # Windows
```

The application will run in the background and listen for the hotkey (Ctrl+Shift+O by default).

## Troubleshooting

### Build Errors
- Ensure Go 1.21+ is installed
- Run `make deps` to update dependencies
- Check that all required system libraries are available

### Runtime Errors
- Verify `.env` file exists and contains valid API key
- Check that the specified model supports vision/image input
- Ensure clipboard and hotkey permissions are granted on your system
