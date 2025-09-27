# Screen OCR LLM

A small desktop utility to select a region of the screen, run OCR via OpenRouter vision models, and copy the result to the clipboard. Runs as a resident tray app with a global hotkey, and also supports a one-shot CLI mode.

## Prerequisites

- Go toolchain installed (go build)
- Windows (current overlay/hotkey path targets Windows)
- OpenRouter API key and a vision-capable model

## Setup

1. Create a `.env` file next to the binary or in the repo root with at least:
   - `OPENROUTER_API_KEY=...`
   - `MODEL=...` (e.g., a vision-capable model supported by OpenRouter)

   Optional keys:
   - `HOTKEY=Ctrl+Alt+q`
   - `ENABLE_FILE_LOGGING=true`
   - `PROVIDERS=providerA,providerB`
   - `OCR_DEADLINE_SEC=15`
   - `SINGLEINSTANCE_PORT_START=49500`
   - `SINGLEINSTANCE_PORT_END=49550`

## Build

- Using Go directly (current platform):
  - `go build -o screen-ocr-llm ./main`

- Using Makefile:
  - Windows GUI binary: `make build-windows` (creates `screen-ocr-llm.exe`)
  - Current platform: `make build`

## Run

- Resident mode (tray + hotkey):
  - Windows: `./screen-ocr-llm.exe`
  - Press `Ctrl+Alt+q` to select a screen region; text is copied to clipboard and shown in a popup.

- One-shot capture (no tray):
  - `./screen-ocr-llm.exe --run-once`
  - If a resident instance is running, the request is delegated; otherwise it runs once and exits.

## Notes

- Logs (if enabled): `screen_ocr_debug.log` with rotation.
- The tool uses a loopback TCP port to ensure a single resident instance and to accept run-once delegations.
