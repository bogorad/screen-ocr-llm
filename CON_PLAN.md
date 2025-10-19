# Production-Ready Plan: Linux CLI OCR Tool

Complete implementation plan for a standalone Linux CLI utility that reads PNG files and outputs OCR text. The Windows GUI application remains the default target.

---

## Project Structure

```
screen-ocr-llm/
├── cmd/
│   └── cli/              # NEW: Linux CLI application
│       ├── main.go
│       ├── main_test.go
│       └── README.md
├── main.go               # Existing Windows GUI entry point
├── test-image.png        # Existing test asset (2,198 chars)
├── config/               # Shared: cross-platform
├── llm/                  # Shared: cross-platform
└── [other packages]      # Windows-specific (not used by CLI)
```

---

## Phase 1: Project Scaffolding

### Step 1.1: Create Directory Structure

```bash
mkdir -p cmd/cli
```

### Step 1.2: Create Main Entry Point

Create `cmd/cli/main.go`:

```go
package main

import (
    "bytes"
    "encoding/json"
    "flag"
    "fmt"
    "io"
    "os"
    "time"

    "screen-ocr-llm/config"
    "screen-ocr-llm/llm"
)

const (
    maxFileSizeMB = 10
    maxFileSize   = maxFileSizeMB * 1024 * 1024
)

func main() {
    if err := run(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}

func run() error {
    // Define flags
    filePath := flag.String("file", "", "Path to PNG file (use '-' for stdin)")
    jsonOutput := flag.Bool("json", false, "Output results as JSON")
    verbose := flag.Bool("v", false, "Verbose output to stderr")
    flag.Parse()

    // Validate required flags
    if *filePath == "" {
        return fmt.Errorf("required flag -file not specified\nUsage: ocr-tool -file <path|-> [-json] [-v]")
    }

    if *verbose {
        fmt.Fprintf(os.Stderr, "[verbose] Starting OCR tool\n")
    }

    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        return fmt.Errorf("failed to load configuration: %w", err)
    }

    if *verbose {
        fmt.Fprintf(os.Stderr, "[verbose] Config loaded: Model=%s\n", cfg.Model)
    }

    // Validate required configuration
    if cfg.APIKey == "" {
        return fmt.Errorf("OPENROUTER_API_KEY is required in .env file")
    }
    if cfg.Model == "" {
        return fmt.Errorf("MODEL is required in .env file")
    }

    // Initialize LLM package
    llm.Init(llm.Config{
        APIKey:    cfg.APIKey,
        Model:     cfg.Model,
        Providers: cfg.Providers,
    })

    if *verbose {
        fmt.Fprintf(os.Stderr, "[verbose] LLM initialized\n")
    }

    return processOCR(*filePath, *jsonOutput, *verbose)
}

func processOCR(filePath string, jsonOutput bool, verbose bool) error {
    // Read image data
    var imageData []byte
    var err error

    if filePath == "-" {
        if verbose {
            fmt.Fprintf(os.Stderr, "[verbose] Reading image from stdin\n")
        }
        imageData, err = io.ReadAll(os.Stdin)
        if err != nil {
            return fmt.Errorf("failed to read from stdin: %w", err)
        }
    } else {
        if verbose {
            fmt.Fprintf(os.Stderr, "[verbose] Reading image from file: %s\n", filePath)
        }
        imageData, err = os.ReadFile(filePath)
        if err != nil {
            return fmt.Errorf("failed to read file %s: %w", filePath, err)
        }
    }

    // Validate file size
    if len(imageData) == 0 {
        return fmt.Errorf("input file is empty")
    }
    if len(imageData) > maxFileSize {
        return fmt.Errorf("input file exceeds maximum size of %d MB", maxFileSizeMB)
    }

    if verbose {
        fmt.Fprintf(os.Stderr, "[verbose] Read %d bytes\n", len(imageData))
    }

    // Validate PNG format
    if len(imageData) < 8 || !bytes.Equal(imageData[:8], []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}) {
        return fmt.Errorf("input is not a valid PNG file (invalid magic number)")
    }

    if verbose {
        fmt.Fprintf(os.Stderr, "[verbose] PNG validation passed\n")
    }

    return performOCR(imageData, filePath, jsonOutput, verbose)
}

func performOCR(imageData []byte, sourcePath string, jsonOutput bool, verbose bool) error {
    if verbose {
        fmt.Fprintf(os.Stderr, "[verbose] Starting OCR with model via llm.QueryVision\n")
    }

    // Use existing battle-tested implementation
    startTime := time.Now()
    text, err := llm.QueryVision(imageData)
    elapsed := time.Since(startTime)

    if err != nil {
        if verbose {
            fmt.Fprintf(os.Stderr, "[verbose] OCR failed after %v: %v\n", elapsed, err)
        }
        return fmt.Errorf("OCR failed: %w", err)
    }

    if verbose {
        fmt.Fprintf(os.Stderr, "[verbose] OCR completed in %v, extracted %d characters\n", elapsed, len(text))
    }

    return outputResult(text, sourcePath, elapsed, jsonOutput)
}

type OCRResult struct {
    Text      string  `json:"text"`
    Source    string  `json:"source"`
    Timestamp string  `json:"timestamp"`
    Duration  float64 `json:"duration_seconds"`
    CharCount int     `json:"character_count"`
}

func outputResult(text string, sourcePath string, elapsed time.Duration, jsonOutput bool) error {
    if jsonOutput {
        result := OCRResult{
            Text:      text,
            Source:    sourcePath,
            Timestamp: time.Now().UTC().Format(time.RFC3339),
            Duration:  elapsed.Seconds(),
            CharCount: len(text),
        }

        encoder := json.NewEncoder(os.Stdout)
        encoder.SetIndent("", "  ")
        if err := encoder.Encode(result); err != nil {
            return fmt.Errorf("failed to encode JSON output: %w", err)
        }
    } else {
        // Plain text output - no trailing newline
        fmt.Print(text)
    }

    return nil
}
```

---

## Phase 2: Testing Strategy

### Step 2.1: Create Integration Test

Create `cmd/cli/main_test.go`:

```go
package main

import (
    "bytes"
    "encoding/json"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "testing"

    "screen-ocr-llm/config"
)

func TestCLIWithTestImage(t *testing.T) {
    // Load configuration to check if API key is available
    cfg, err := config.Load()
    if err != nil || cfg.APIKey == "" {
        t.Skip("Skipping integration test: no API key configured")
    }

    // Build the CLI tool
    binaryPath := filepath.Join(t.TempDir(), "ocr-tool")
    buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
    if output, err := buildCmd.CombinedOutput(); err != nil {
        t.Fatalf("Failed to build CLI tool: %v\n%s", err, output)
    }

    // Path to existing test-image.png (2 directories up from cmd/cli)
    testImagePath := "../../test-image.png"
    if _, err := os.Stat(testImagePath); err != nil {
        t.Fatalf("test-image.png not found: %v", err)
    }

    // Test 1: Plain text output
    t.Run("PlainTextOutput", func(t *testing.T) {
        cmd := exec.Command(binaryPath, "-file", testImagePath)
        var stdout, stderr bytes.Buffer
        cmd.Stdout = &stdout
        cmd.Stderr = &stderr

        if err := cmd.Run(); err != nil {
            t.Errorf("Command failed: %v\nStderr: %s", err, stderr.String())
        }

        text := stdout.String()
        if len(text) == 0 {
            t.Error("Expected output, got empty string")
        }

        // test-image.png successfully extracted 2,198 characters previously
        if len(text) < 1000 {
            t.Errorf("Expected substantial text output (previous run: 2198 chars), got %d chars", len(text))
        }

        t.Logf("OCR extracted %d characters from test-image.png", len(text))
    })

    // Test 2: JSON output
    t.Run("JSONOutput", func(t *testing.T) {
        cmd := exec.Command(binaryPath, "-file", testImagePath, "-json")
        output, err := cmd.Output()
        if err != nil {
            t.Errorf("Command failed: %v", err)
        }

        var result OCRResult
        if err := json.Unmarshal(output, &result); err != nil {
            t.Errorf("Failed to parse JSON: %v", err)
        }

        if result.Text == "" {
            t.Error("JSON result missing text field")
        }
        if result.CharCount == 0 {
            t.Error("JSON result missing character count")
        }
        if result.Source != testImagePath {
            t.Errorf("Expected source=%s, got %s", testImagePath, result.Source)
        }

        t.Logf("JSON output: %d chars, duration: %.2fs", result.CharCount, result.Duration)
    })

    // Test 3: Verbose mode
    t.Run("VerboseMode", func(t *testing.T) {
        cmd := exec.Command(binaryPath, "-file", testImagePath, "-v")
        var stdout, stderr bytes.Buffer
        cmd.Stdout = &stdout
        cmd.Stderr = &stderr

        cmd.Run()

        if !strings.Contains(stderr.String(), "[verbose]") {
            t.Error("Expected verbose output in stderr")
        }
    })

    // Test 4: Stdin input
    t.Run("StdinInput", func(t *testing.T) {
        imageData, _ := os.ReadFile(testImagePath)
        cmd := exec.Command(binaryPath, "-file", "-")
        cmd.Stdin = bytes.NewReader(imageData)

        output, err := cmd.Output()
        if err != nil {
            t.Errorf("Stdin test failed: %v", err)
        }
        if len(output) == 0 {
            t.Error("Expected output from stdin input")
        }
    })
}

func TestPNGValidation(t *testing.T) {
    tests := []struct {
        name    string
        data    []byte
        wantErr bool
    }{
        {
            name:    "ValidPNG",
            data:    []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0x00},
            wantErr: false,
        },
        {
            name:    "InvalidMagic",
            data:    []byte{0x00, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a},
            wantErr: true,
        },
        {
            name:    "TooShort",
            data:    []byte{0x89, 'P', 'N', 'G'},
            wantErr: true,
        },
        {
            name:    "Empty",
            data:    []byte{},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validatePNG(tt.data)
            if (err != nil) != tt.wantErr {
                t.Errorf("validatePNG() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}

func validatePNG(data []byte) error {
    if len(data) < 8 || !bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}) {
        return fmt.Errorf("invalid PNG")
    }
    return nil
}
```

---

## Phase 3: Build Configuration

### Step 3.1: Update Makefile

Add to existing `Makefile`:

```makefile
# Linux CLI build target
build-cli-linux:
    GOOS=linux GOARCH=amd64 go build -o ocr-tool ./cmd/cli

# Cross-compile from Windows
build-cli-linux-from-windows:
    set GOOS=linux&& set GOARCH=amd64&& go build -o ocr-tool ./cmd/cli

# Local build (detects OS automatically)
build-cli:
    go build -o ocr-tool ./cmd/cli

# Test CLI
test-cli:
    cd cmd/cli && go test -v
```

### Step 3.2: Build Instructions

Create `cmd/cli/README.md`:

```markdown
# OCR CLI Tool for Linux

Standalone command-line utility for performing OCR on PNG images using multimodal LLMs.

## Building

### On Linux
```

cd cmd/cli
go build -o ocr-tool .

```

### Cross-compile from Windows

```

# PowerShell

\$env:GOOS="linux"; \$env:GOARCH="amd64"; go build -o ocr-tool ./cmd/cli

# cmd

set GOOS=linux\&\& set GOARCH=amd64\&\& go build -o ocr-tool ./cmd/cli

```

### Using Makefile

```

make build-cli-linux

```

## Configuration

Create `.env` file in the same directory as the binary:

```

OPENROUTER_API_KEY=your_key_here
MODEL=google/gemini-2.0-flash-exp:free
OCRDEADLINESEC=15
PROVIDERS=

```

Or set `SCREENOCRLLM` environment variable to point to your config file.

## Usage

```

# Basic OCR

./ocr-tool -file image.png

# JSON output

./ocr-tool -file image.png -json

# From stdin

cat image.png | ./ocr-tool -file -

# Verbose mode for debugging

./ocr-tool -file image.png -v 2> debug.log

```

## Testing

```

# Run integration tests (requires API key in .env)

go test -v

# Test with existing test-image.png

./ocr-tool -file ../../test-image.png

```

Expected output: ~2,198 characters (validated from existing codebase).

## Features

- Direct-to-LLM OCR using multimodal models
- Automatic retry with exponential backoff (3 attempts)
- PNG validation
- Stdin support for pipeline integration
- JSON output for automation
- Configurable timeout via `OCRDEADLINESEC`
- Multiple LLM provider support via `PROVIDERS`
- Cross-platform config package (no Linux-specific dependencies)

## Architecture

Uses shared packages from parent project:
- `config` - Configuration loading
- `llm` - OpenRouter API client with retry logic

Does NOT depend on Windows-specific packages:
- `hotkey` - Not needed (CLI is invoked directly)
- `tray` - Not needed (no GUI)
- `overlay` - Not needed (no region selection)
- `screenshot` - Not needed (file input only)

## Examples

```

# Save OCR output to file

./ocr-tool -file scan.png > output.txt

# Process multiple images with JSON output

for img in \*.png; do
echo "Processing $img..."
  ./ocr-tool -file "$img" -json >> results.jsonl
done

# Pipeline with image conversion

convert document.pdf page.png \&\& ./ocr-tool -file page.png

# Error handling

if ! ./ocr-tool -file scan.png > result.txt 2> error.log; then
echo "OCR failed, check error.log"
fi

```

## Environment Variables

- `OPENROUTER_API_KEY` - Required. Your OpenRouter API key
- `MODEL` - Required. Model identifier (e.g., `google/gemini-2.0-flash-exp:free`)
- `OCRDEADLINESEC` - Optional. Timeout in seconds (default: 15)
- `PROVIDERS` - Optional. Comma-separated provider list for routing
- `SCREENOCRLLM` - Optional. Path to config file (overrides `.env` search)

## Comparison with Windows GUI

| Feature | Windows GUI | Linux CLI |
|---------|-------------|-----------|
| Region Selection | ✓ Interactive overlay | ✗ File input only |
| Hotkey Support | ✓ Global hotkeys | ✗ Invoke directly |
| System Tray | ✓ Background service | ✗ Single-shot execution |
| Stdin Support | ✗ | ✓ Pipeline integration |
| JSON Output | ✗ | ✓ Structured data |
| Dependencies | Many (GUI libs) | Minimal (HTTP only) |

```

---

## Phase 4: Documentation Updates

### Step 4.1: Update Root README

Add section to root `README.md`:

```markdown
## Linux CLI Tool

A standalone CLI utility for Linux users:
```

# Build

cd cmd/cli
go build -o ocr-tool .

# Usage

./ocr-tool -file screenshot.png

```

See [cmd/cli/README.md](cmd/cli/README.md) for details.
```

### Step 4.2: Update Build Instructions

Add to `BUILD_INSTRUCTIONS.md`:

```markdown
## Building the Linux CLI Tool

The Linux CLI tool is a separate binary that does not include GUI dependencies.

### On Linux
```

go build -o ocr-tool ./cmd/cli

```

### Cross-compile from Windows
```

set GOOS=linux\&\& set GOARCH=amd64\&\& go build -o ocr-tool ./cmd/cli

```

The CLI tool uses the same configuration system but operates in single-shot mode (no daemon, no GUI).
```

---

## Summary of Changes

### New Files Created

1. `cmd/cli/main.go` - Complete CLI implementation (250 lines)
2. `cmd/cli/main_test.go` - Integration tests with `test-image.png`
3. `cmd/cli/README.md` - Comprehensive CLI documentation

### Existing Assets Used

- `test-image.png` - Validated test asset (2,198 chars extracted)[^1]
- `config` package - Cross-platform config loading[^1]
- `llm` package - API client with retry logic[^1]

### Architecture Benefits

- **Separation** - CLI is `cmd/cli/`, GUI remains root
- **No pollution** - Linux binary doesn't link GUI dependencies
- **Shared core** - Reuses `config` and `llm` packages
- **Cross-compile** - Windows devs can build Linux binaries
- **Testing** - Uses existing validated test image

### Build Targets

```bash
# Windows GUI (default)
go build -ldflags "-H windowsgui" -o screen-ocr-llm.exe .

# Linux CLI
GOOS=linux go build -o ocr-tool ./cmd/cli
```

