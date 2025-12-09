# OCR CLI Tool for Linux

Standalone command-line utility for performing OCR on PNG images using multimodal LLMs.

## Building

### On Linux
```
cd src/cmd/cli
go build -o ocr-tool .
```

### Cross-compile from Windows

```
# PowerShell

$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o ocr-tool ./src/cmd/cli

# cmd

set GOOS=linux&& set GOARCH=amd64&& go build -o ocr-tool ./src/cmd/cli

```

### Using Makefile

```
make build-cli-linux

```

## Configuration

The CLI tool loads the OpenRouter API key from multiple sources in this priority order:

### 1. Secret File (Production/Kubernetes)
Path: `/run/secrets/api_keys/openrouter`

For containerized deployments with SOPS or Kubernetes secrets:
```
# Kubernetes secret mount (managed by your cluster)
# No manual configuration needed - mounted by deployment
```

### 2. Environment Variable (Development)
```
export OPENROUTER_API_KEY=sk-or-v1-your-key-here
./ocr-tool -file image.png
```

### 3. Config File (Local Development)
Create a `.env` file in the same directory as the binary:
```
OPENROUTER_API_KEY=your_key_here
MODEL=google/gemini-2.0-flash-exp:free
OCRDEADLINESEC=15
PROVIDERS=
```

Or set `SCREENOCRLLM` environment variable to point to your config file:
```
export SCREENOCRLLM=/path/to/your/.env
```

### Configuration Hierarchy
The API key search stops at the first successful match:
- ✓ Found in `/run/secrets/api_keys/openrouter` → Use this
- ✗ Not found → Check `OPENROUTER_API_KEY` env var
- ✗ Not found → Check `.env` file
- ✗ Not found → Exit with error

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

./ocr-tool -file ../../../test-image.png

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

Uses shared packages from parent directory (../../):
- `../config` - Configuration loading (screen-ocr-llm/src/config)
- `../llm` - OpenRouter API client with retry logic (screen-ocr-llm/src/llm)

Does NOT depend on Windows-specific packages:
- `../hotkey` - Not needed (CLI is invoked directly)
- `../tray` - Not needed (no GUI)
- `../overlay` - Not needed (no region selection)
- `../screenshot` - Not needed (file input only)

## Examples

```
# Save OCR output to file

./ocr-tool -file scan.png > output.txt

# Process multiple images with JSON output

for img in *.png; do
echo "Processing $img..."
  ./ocr-tool -file "$img" -json >> results.jsonl
done

# Pipeline with image conversion

convert document.pdf page.png && ./ocr-tool -file page.png

# Error handling

if ! ./ocr-tool -file scan.png > result.txt 2> error.log; then
echo "OCR failed, check error.log"
fi

```

## Kubernetes Deployment with SOPS

Example deployment manifest that mounts SOPS-encrypted secrets:

```
apiVersion: v1
kind: Pod
metadata:
  name: ocr-tool
spec:
  containers:
  - name: ocr-tool
    image: your-registry/ocr-tool:latest
    command: ["/ocr-tool", "-file", "/input/image.png"]
    env:
    - name: MODEL
      value: "google/gemini-2.0-flash-exp:free"
    volumeMounts:
    - name: api-secrets
      mountPath: /run/secrets/api_keys
      readOnly: true
    - name: input
      mountPath: /input
  volumes:
  - name: api-secrets
    secret:
      secretName: openrouter-api-key
      items:
      - key: openrouter
        path: openrouter
        mode: 0600
  - name: input
    hostPath:
      path: /path/to/images
```

Create the secret with SOPS:
```
# Encrypt your API key with SOPS
echo "sk-or-v1-your-key-here" | sops encrypt /dev/stdin > openrouter.enc

# Create Kubernetes secret from encrypted file
kubectl create secret generic openrouter-api-key \
  --from-file=openrouter=openrouter.enc \
  --dry-run=client -o yaml | kubectl apply -f -
```

## Environment Variables

- `OPENROUTER_API_KEY` - Optional. Your OpenRouter API key (checked after secret file)
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