# Screen OCR LLM

Inspired by [the original code](https://github.com/cherjr/screen-ocr-llm)

A small desktop utility to select a region of the screen, run OCR via OpenRouter vision models, and copy the result to the clipboard.

For the latest release details, see `docs/releases/2.6.0.md`.

## Architecture Overview

### Directory Structure

- `src/` - All source code packages
- `tests/` - Integration and debug test files
- Unit tests (`*_test.go`) remain with their respective packages

At a high level, the Windows app is structured as:

- `src/main`:
  - Parses Cobra flags (`--run-once`, `--api-key-path`, `--default-mode`) and keeps compatibility with legacy single-dash forms.
  - Ensures single resident instance via a TCP preflight on the configured port.
  - Uses shared runtime bootstrap (`src/runtimeinit`) to load config, set logging, initialize OCR/LLM/clipboard dependencies, and perform startup ping checks.
  - Enables DPI awareness and monitor diagnostics via Windows-specific helpers.
  - In resident mode:
    - Starts the central event loop (`src/eventloop`), the system tray (`src/tray`), and global hotkey listener (`src/hotkey`).
  - In `--run-once` mode:
    - First tries to delegate to a running resident via `src/singleinstance.Client`.
    - If no resident is available, runs a standalone capture+OCR flow.

- `src/eventloop`:
  - Owns the single-instance TCP server (`src/singleinstance.Server`).
  - Listens for:
    - Global hotkey triggers.
    - Delegated `--run-once` requests.
  - For each request:
    - Uses `src/overlay.Selector`/`src/gui` to run the interactive region selector.
    - Submits OCR work to a bounded worker pool (`src/worker` + `src/ocr` + `src/llm`).
    - Routes completion through shared result-target behavior (clipboard/stdout/delegated response), updates UI via `src/popup`, and preserves busy semantics.
  - Enforces that only one OCR job runs at a time ("busy" behavior).

- `src/session`:
  - Provides a shared OCR session executor for region selection, countdown popup lifecycle, OCR with deadline, and pluggable output targets.
  - Used by standalone run-once fallback and shared with resident-related result target logic.

- `src/overlay` + `src/gui` + `src/screenshot`:
  - Implement the Windows overlay window with rectangle/lasso region selection.
  - Capture the selected region (multi-monitor aware) as PNG bytes for OCR.
  - In lasso mode, capture the bounding rectangle and fill outside-polygon pixels with white before OCR.

- `src/llm` + `src/ocr`:
  - `src/ocr` captures the region and forwards it to `src/llm`.
  - `src/llm` calls the OpenRouter Chat Completions API with a strict OCR-style prompt and optional `PROVIDERS` routing.

- `src/singleinstance`:
  - TCP-based discovery and delegation so `--run-once` clients hand work to the resident when available.

- `src/tray` + `src/popup` + `src/notification`:
  - System tray icon, About/Exit menu, and small non-intrusive popups.
  - Countdown popup appears during OCR and is updated/closed when results arrive.

The Linux CLI (`src/cmd/cli`) is a separate, GUI-free binary that reuses `src/config` and `src/llm` to run OCR on PNG input (file or stdin) and print plain-text or JSON output.

## Prerequisites

- Go toolchain installed (go build) if you want to build yourself; otherwise use .exe from releases.
- Windows (current overlay/hotkey path targets Windows)
- OpenRouter API key and a vision-capable model (`:free` models are also supported, but not recommended)

## Linux CLI Tool

A standalone CLI utility for Linux users:

```sh
# Build
cd src/cmd/cli
go build -o ocr-tool .

# Usage
./ocr-tool --file screenshot.png
```

See [src/cmd/cli/README.md](src/cmd/cli/README.md) for details.

## Setup

1.  Create a `.env` file in the same directory as the executable with the following required keys:
    - `OPENROUTER_API_KEY=`
    - `MODEL=` (vision-capable model, e.g., `qwen/qwen3-vl-235b-a22b-instruct`)
    - Optional: `OPENROUTER_API_KEY_FILE=` (default key-file path is `/run/secrets/api_keys/openrouter`)

    Alternatively, you can set each of these as an environment variable.

2.  Alternatively, you can point the app to a config file via an environment variable:
    - Set `SCREEN_OCR_LLM` to the full path of a `.env`-format file. If `.env` is not found in the executable directory, the app will load configuration from this path.

3.  You can also add these optional keys to your `.env` file to customize behavior:
    - `HOTKEY=Ctrl+Alt+q`
      - Supported modifiers: `Ctrl`, `Alt`, `Shift`, `Win/Cmd/Super`
      - Supported keys: `A-Z`, `0-9`, `F1-F24`, and common special keys
      - Example: `HOTKEY=F13`
    - `ENABLE_FILE_LOGGING=true`
    - `PROVIDERS=providerA,providerB`
    - `OCR_DEADLINE_SEC=20` (default is 20 seconds if unset)
    - `DEFAULT_MODE=rectangle` (accepted: `rect`, `rectangle`, `lasso`; default is rectangle)
    - `SINGLEINSTANCE_PORT_START=49500`
    - `SINGLEINSTANCE_PORT_END=49550`

## Configuration and Precedence

Detailed source resolution and precedence rules are documented in `docs/README.md` under `Runtime Configuration Sources and Precedence`.

## Build

- **Using Go directly**:
  - On Windows (no console window):
    ```sh
    go build -ldflags "-H=windowsgui" -o screen-ocr-llm.exe ./src/main
    ```
  - On Linux/macOS:
    ```sh
    go build -o screen-ocr-llm ./src/main
    ```

- **Using the Makefile** (for a Windows GUI binary):
  ```sh
  make build-windows
  ```
  This creates a `screen-ocr-llm.exe` file that runs without a console window.

## Execution Modes

The application offers two primary modes of operation:

### Resident Mode (Default)

This is the standard mode for continuous, everyday use. The application runs quietly in the background, accessible via a system tray icon and a global hotkey.

- **How to run**: Execute the binary without any command-line flags.
  ```sh
  ./screen-ocr-llm.exe
  ```
- **Functionality**:
  - Manages a system tray icon with "About" and "Exit" options.
  - Listens for a global hotkey (default: `Ctrl+Alt+q`) to start a screen capture.
  - In the selection overlay, drag for rectangle mode, press `Space` to toggle lasso mode, and press `Esc` to cancel.
  - In lasso mode, complete the selection by releasing the mouse near the start point to close the loop.
  - After a region is selected, the extracted text is automatically copied to your clipboard and shown in a brief popup notification. Lasso captures are still sent as rectangular images, with pixels outside the lasso filled solid white.
  - It ensures that only one instance of the application is running at any time.

### One-Shot Mode (`--run-once`)

This mode is intended for single, on-demand captures initiated from the command line or within scripts.

- **How to run**: Execute the binary using the `--run-once` flag.
  ```sh
  ./screen-ocr-llm.exe --run-once
  ```
- **Supported arguments**:
  - `--run-once`
  - `--api-key-path <path>`
  - `--default-mode <rect|rectangle|lasso>`
  - Legacy compatibility: single-dash long forms (`-run-once`, `-api-key-path`, `-default-mode`)
- **Optional key path override**:
  ```sh
  ./screen-ocr-llm.exe --run-once --api-key-path /run/secrets/api_keys/openrouter_key
  ```
- **Optional initial selection mode override**:
  ```sh
  ./screen-ocr-llm.exe --run-once --default-mode lasso
  ```
- **Functionality**:
  - Bypasses the system tray and immediately prompts you to select a region on the screen (same rectangle/lasso controls as resident mode).
  - Copies the resulting text to the clipboard.
  - Exits silently as soon as the capture and OCR process is finished.

### Combined Operation (Delegation)

The two modes are designed to work together intelligently to prevent conflicts and ensure smooth operation.

- When you start a new capture with `--run-once`, the application first checks if a **resident** instance is already running.
- **If a resident instance is found**, the `--run-once` process delegates the capture request to the running instance and exits. The resident application then takes over, presenting the screen selection UI.
- If `--api-key-path` is provided on a delegated `--run-once` client, the client still delegates and the resident instance configuration remains authoritative.
- If `--default-mode` is provided on a delegated `--run-once` client, the client still delegates and the resident instance configuration remains authoritative.
- **If no resident instance is active**, the `--run-once` process will handle the capture itself in a temporary standalone mode before exiting.
- **Startup validation**: On launch, the app performs a minimal LLM connectivity check (1-token ping). If it fails, a blocking error dialog is shown and the app exits. In `--run-once`, if a resident is detected and the request is delegated, the client does not ping.
- **High-DPI**: The app enables DPI awareness and uses the full virtual screen for overlays and screenshots to work correctly on scaled multi-monitor setups.
- **Logging**: Controlled by `ENABLE_FILE_LOGGING`. When `false`, logs are suppressed; when `true`, logs are written to `screen_ocr_debug.log` (size-rotated). In GUI builds, stdout/stderr are hidden, so enable file logging for diagnostics.

This delegation mechanism ensures a stable and predictable user experience by guaranteeing that only one screen selection process can be active at a time.

## Notes

- **Logging**: Controlled by `ENABLE_FILE_LOGGING`. When `false`, logs are suppressed; when `true`, logs are written to `screen_ocr_debug.log` with size-based rotation. In GUI builds, stdout/stderr are hidden, so enable file logging for diagnostics.
- **Single Instance**: The tool uses a loopback TCP port to enforce a single resident instance and to manage delegation from `--run-once` clients.
- **Configuration precedence**: See `Configuration and Precedence` above for `.env`, CLI, and delegation behavior.
