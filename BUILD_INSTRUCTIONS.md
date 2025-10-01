# Build Instructions

## Quick Build

### Windows
```cmd
build.cmd
```
or
```cmd
go build -o screen-ocr-llm.exe ./main
```

### Linux/macOS
```bash
make build
```
or
```bash
go build -o screen-ocr-llm ./main
```

## Important Notes

### ⚠️ Always Build from `./main` Directory

The application's main package is in the `./main` directory. **Always** specify `./main` when building:

```bash
# ✅ CORRECT
go build -o screen-ocr-llm.exe ./main

# ❌ WRONG - Will build test files instead!
go build -o screen-ocr-llm.exe
```

If you run `go build` without specifying `./main`, it may build other `package main` files in the root directory (like test files), resulting in the wrong executable.

### Test Files

Test files with `package main` (like `test_clipboard.go`) should be:
- Kept in a separate directory (e.g., `tests/`)
- Or renamed with `_test.go` suffix if they're actual Go tests
- Or given a different package name

## Build Options

### Standard Build (with console window)
```bash
go build -o screen-ocr-llm.exe ./main
```

### Windows GUI Build (no console window)
```bash
go build -ldflags "-H=windowsgui" -o screen-ocr-llm.exe ./main
```

### Cross-Platform Builds

See the `Makefile` for cross-platform build targets:
- `make build-windows` - Windows (GUI mode)
- `make build-macos` - macOS Intel
- `make build-macos-arm` - macOS Apple Silicon
- `make build-linux` - Linux

## Running the Application

After building:

### Windows
```cmd
.\screen-ocr-llm.exe
```

### Linux/macOS
```bash
./screen-ocr-llm
```

The application will:
1. Load configuration from `.env`
2. Start as a system tray application
3. Listen for the configured hotkey (default: Ctrl+Alt+Q, or Ctrl+Win+E from your .env)
4. Show a system tray icon

## Troubleshooting

### "Clipboard test" runs instead of the app
**Problem:** You built without specifying `./main`

**Solution:** Use `build.cmd` or `go build -o screen-ocr-llm.exe ./main`

### Hotkey not working
**Problem:** The hotkey from `.env` is not being detected

**Solution:** 
1. Check your `.env` file has the correct `HOTKEY=` setting
2. Rebuild with `build.cmd` or `go build -o screen-ocr-llm.exe ./main`
3. Restart the application
4. Check `screen_ocr_debug.log` for the loaded hotkey configuration

### Application doesn't start
**Problem:** Missing dependencies or API key

**Solution:**
1. Ensure `.env` file exists with valid `OPENROUTER_API_KEY`
2. Run `go mod tidy` to install dependencies
3. Check `screen_ocr_debug.log` for error messages

## Development

### Run Tests
```bash
go test ./...
```

### Run Tests with Coverage
```bash
go test -cover ./...
```

### Format Code
```bash
go fmt ./...
```

### Vet Code
```bash
go vet ./...
```

### All Checks
```bash
make check
```

## Configuration

Edit `.env` to configure:
- `OPENROUTER_API_KEY` - Your OpenRouter API key
- `MODEL` - The LLM model to use
- `HOTKEY` - Custom hotkey combination (e.g., `Ctrl+Win+E`, `Ctrl+Alt+Q`)

Supported hotkey modifiers: Ctrl, Alt, Shift, Win/Cmd/Super
Supported keys: A-Z, 0-9, F1-F12, and common special keys

## Logging

Enable file logging in `.env`:
```
ENABLE_FILE_LOGGING=true
```

Logs will be written to `screen_ocr_debug.log` in the application directory.

