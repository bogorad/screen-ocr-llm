## Screen OCR LLM - Architecture (Current Implementation)

Version: 2.0
Status: Up-to-date with current codebase
Owner: Engineering
Scope: Document the actual runtime architecture, concurrency model, IPC, and data flow as implemented in the Go codebase.

---

### 1) Overview

Screen OCR LLM is a Windows-focused desktop utility that lets users select a screen region, performs OCR using OpenRouter vision models, and copies the result to the clipboard or returns it to a CLI client. The app runs as a resident process with a system tray icon and global hotkey, and also supports a one-shot CLI mode.

---

### 2) Execution Modes

- Resident (default): Starts tray, hotkey listener, single-instance TCP server, and an event loop coordinating selections and OCR.
- Run once: `--run-once` captures one region and copies text to clipboard; exits. Delegates to resident if available, else runs standalone.

---

### 3) High-Level Components (packages)

- main: entrypoint, config/logging init, resident lifecycle, CLI one-shot fallback.
- eventloop: single-threaded coordinator for hotkey events and run-once TCP requests. Owns overlay selection, busy state, deadlines, and result handling.
- singleinstance: loopback TCP resident server/client for delegation and single-instance detection.
- overlay: synchronous API to obtain a region selection; Windows-specific adapter over gui.
- gui: region selection UI and some legacy helpers used by overlay and hotkey.
- hotkey: global key detection (Ctrl+Alt+Q) via gohook; also contains a legacy end-to-end OCR path.
- worker: bounded OCR worker pool (size = NumCPU) with a 1-slot input queue for back-pressure and deadline-aware execution.
- screenshot: captures screen region as PNG bytes.
- ocr: wraps the LLM vision call; optional debug image dump.
- llm: OpenRouter client (vision) with optional provider preferences.
- clipboard: mutex-guarded clipboard writes.
- popup/notification/tray/logutil/config: UI feedback, system tray, logging rotation, and env-driven configuration.
- router/messages: legacy message bus types (present but not used by the current event loop path).

---

### 4) Concurrency Model

- Single event-loop goroutine in eventloop processes:
  - Hotkey events (posted via an internal channel)
  - Incoming run-once TCP connections (delegated clients)
  - OCR completion callbacks (posted back into the loop)
- OCR work runs off-loop in worker pool goroutines.
- Back-pressure: worker queue capacity is 1; submissions when full are dropped with user-visible feedback ("Busy, please retry").
- Clipboard writes are serialized via a short-lived mutex.

---

### 5) Single-Instance & IPC (TCP)

- Transport: loopback TCP on 127.0.0.1; default port range [49500, 49550] configured via env:
  - SINGLEINSTANCE_PORT_START, SINGLEINSTANCE_PORT_END
- Server: eventloop starts a tcpServer and binds ONLY the start port; pre-flight in main ensures uniqueness by probing that port.
- Client: run-once mode scans the range for a resident (PING/PONG) and then sends a request.
- Protocol:
  - Handshake: client sends a single line, either `STDOUT\n` or `CLIPBOARD\n`.
  - Success response: `SUCCESS\n` followed by optional text (for STDOUT mode).
  - Error response: `ERROR\n` followed by a human-readable message.
  - Health probe: `PING\n` -> `PONG\n`.

---

### 6) Event Flow (Resident)

1. Hotkey detected -> eventloop.handleHotkey:
   - If busy: show popup "Busy, please retry".
   - Else: open overlay selection, obtain region synchronously.
   - Submit OCR job with deadline OCR_DEADLINE_SEC (default 15s).
2. Run-once TCP request -> eventloop.handleConn:
   - If busy: immediate error.
   - Else: selection -> submit OCR -> on completion:
     - If STDOUT mode: return text and also show popup.
     - If CLIPBOARD mode: write to clipboard and return success.
3. Completion -> eventloop.handleResult:
   - Manages clipboard/popup and resets busy state.

Note: The hotkey package currently contains a legacy path that directly performs OCR and clipboard within its own callback using gui and ocr packages. The event loop also registers a channel-based hotkey handler. In practice, the overlay path used by eventloop relies on gui; this is a transitional state to a fully eventloop-driven hotkey.

---

### 7) Worker Pool and Deadlines

- Size: runtime.NumCPU() workers.
- Queue: 1 slot (strict back-pressure); dropped submissions show a "Busy..." popup.
- Deadline: env `OCR_DEADLINE_SEC` (default 15). Cancellation is honored by a deadline-aware shim that returns ctx.Err() on timeout.

---

### 8) UI/UX

- Tray: getlantern/systray with About and Exit. Tooltip updates reflect idle vs processing state; About dialog can display resident port info.
- Overlay: synchronous region selection via Windows adapter over gui.
- Popup/Notification: shows the recognized text (truncated for notifications). In resident paths, popup is shown after clipboard write; in stdout mode it is also shown for visibility.

---

### 9) Configuration and Logging

- Config (config.Load): .env resolution from cwd or executable dir.
- Keys:
  - OPENROUTER_API_KEY, MODEL (required)
  - HOTKEY (default "Ctrl+Alt+q")
  - ENABLE_FILE_LOGGING (true|false)
  - PROVIDERS (comma-separated for OpenRouter provider order)
  - SINGLEINSTANCE_PORT_START/END, OCR_DEBUG_SAVE_IMAGES, OCR_DEADLINE_SEC
- Logging (logutil): size-rotated file logging to screen_ocr_debug.log (10MB, 3 archives). API key redaction helper provided.

---

### 10) Error Handling & Reliability

- Event-loop handlers are lightweight; OCR is offloaded.
- Long-running operations are wrapped with context deadlines.
- Busy state prevents concurrent overlays and provides user feedback.
- Tray tooltip conveys processing state; About dialog includes runtime info where applicable.

---

### 11) Testing Notes

- Unit tests cover core packages (hotkey, ocr, llm, screenshot, singleinstance, worker).
- Recommended checks: `go test -race ./...`. Stress tests for multiple run-once clients should observe no concurrent overlays and correct busy signaling.

---

### 12) Known Legacy/Transitional Areas

- router/ and messages/ remain in the tree but are not used by the resident event-loop architecture.
- hotkey package still contains an older flow that directly performs OCR; the target state is to treat hotkey purely as a signal into the event loop.

---

### 13) Future Work (nice-to-have)

- Consolidate hotkey to purely signal the event loop; remove the direct OCR path.
- Remove unused router/messages modules after full migration.
- Optional pprof instrumentation and soak tests for handler latency.
