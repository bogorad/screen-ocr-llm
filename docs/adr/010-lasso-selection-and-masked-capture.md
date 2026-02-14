# ADR-010: Lasso Selection Mode and Masked OCR Capture

## Status

Accepted

## Date

2026-02-14

## Context

Rectangle selection was fast and reliable, but users needed a way to isolate irregular text regions without including nearby clutter.

The OCR and LLM pipeline currently expects a rectangular image payload. Introducing free-form selection without a clear contract would risk changing existing behavior across resident mode, delegated run-once requests, and standalone run-once fallback.

We also observed resident-mode keyboard delivery differences where overlay `WM_KEYDOWN` for `Space` and `Escape` could be suppressed while global hotkey hooks were active.

## Decision

Implement a dual-mode region selector on Windows with an explicit capture contract:

### Selection modes
- Keep **rectangle mode** as the default behavior.
- Add **lasso mode** that can be toggled with `Space`.
- Require lasso completion by releasing the mouse near the starting point (close-loop rule).

### Capture contract
- Keep OCR/LLM payloads as rectangular PNG images.
- Extend region metadata with optional polygon points in virtual-screen coordinates.
- For lasso selections, capture the polygon bounding rectangle and fill pixels outside the polygon with solid white before OCR.

### Configuration and overrides
- Add `DEFAULT_MODE` config for initial selector mode.
- Support `--default-mode` CLI override with normalized values: `rect`, `rectangle`, `lasso`.
- Keep resident configuration authoritative when run-once requests are delegated.

### Input reliability
- Keep existing `WM_KEYDOWN` handling.
- Add async key polling fallback (`GetAsyncKeyState`) for `Space` and `Escape` to handle resident-mode edge cases.
- Apply toggle debounce and edge detection to avoid missed taps and rapid double toggles.

## Consequences

### Positive
- Users can target irregular screen regions with less OCR noise.
- Rectangle-mode behavior and single-instance delegation contract remain intact.
- Resident and run-once flows stay aligned under one selection model.

### Negative
- Overlay implementation complexity increases (mode state, path geometry, masking, cursor handling).
- Additional Windows-specific logic is required for keyboard reliability.

### Neutral
- Payloads remain rectangular at API boundaries, preserving existing OCR/LLM integration.
- Lasso behavior is opt-in per session or configurable default.

## References

- **ADR-001**: Callback to Direct Return Refactoring
- **ADR-006**: DPI Awareness Implementation
- **ADR-007**: TCP-based Single Instance
- **ADR-009**: Multi-Monitor Support and Coordinate Handling
