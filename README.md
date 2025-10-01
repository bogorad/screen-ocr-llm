# Screen OCR LLM

\*\* Inspured by [the original code](https://github.com/cherjr/screen-ocr-llm)

A small desktop utility to select a region of the screen, run OCR via OpenRouter vision models, and copy the result to the clipboard.

## Prerequisites

- Go toolchain installed (go build)
- Windows (current overlay/hotkey path targets Windows)
- OpenRouter API key and a vision-capable model

## Setup

1.  Create a `.env` file in the same directory as the executable with the following required keys:
    - `OPENROUTER_API_KEY=`
    - `MODEL=` (e.g., `google/gemma-2-9b-it`)

2.  You can also add these optional keys to your `.env` file to customize behavior:
    - `HOTKEY=Ctrl+Alt+q`
    - `ENABLE_FILE_LOGGING=true`
    - `PROVIDERS=providerA,providerB`
    - `OCR_DEADLINE_SEC=15`
    - `SINGLEINSTANCE_PORT_START=49500`
    - `SINGLEINSTANCE_PORT_END=49550`

## Build

- **Using Go directly** (for your current platform):

  ```sh
  go build -o screen-ocr-llm ./main
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
  - After a region is selected, the extracted text is automatically copied to your clipboard and shown in a brief popup notification.
  - It ensures that only one instance of the application is running at any time.

### One-Shot Mode (`--run-once`)

This mode is intended for single, on-demand captures initiated from the command line or within scripts.

- **How to run**: Execute the binary using the `--run-once` flag.
  ```sh
  ./screen-ocr-llm.exe --run-once
  ```
- **Functionality**:
  - Bypasses the system tray and immediately prompts you to select a region on the screen.
  - Copies the resulting text to the clipboard.
  - Exits silently as soon as the capture and OCR process is finished.

### Combined Operation (Delegation)

The two modes are designed to work together intelligently to prevent conflicts and ensure smooth operation.

- When you start a new capture with `--run-once`, the application first checks if a **resident** instance is already running.
- **If a resident instance is found**, the `--run-once` process delegates the capture request to the running instance and exits. The resident application then takes over, presenting the screen selection UI.
- **If no resident instance is active**, the `--run-once` process will handle the capture itself in a temporary standalone mode before exiting.

This delegation mechanism ensures a stable and predictable user experience by guaranteeing that only one screen selection process can be active at a time.

## Notes

- **Logging**: If enabled in the `.env` file, logs are written to `screen_ocr_debug.log`, which is automatically rotated based on size.
- **Single Instance**: The tool uses a loopback TCP port to enforce a single resident instance and to manage delegation from `--run-once` clients.
