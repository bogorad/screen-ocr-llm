# ADR-009: Multi-Monitor Support and Coordinate Handling

## Status

Accepted

## Date

2025-12-14

## Context

The application experienced significant issues on multi-monitor Windows setups:

- **Cursor not appearing**: The selection crosshairs failed to display when a second monitor was connected, showing a circle cursor instead.
- **Incorrect capture areas**: Selections on monitors positioned above or below the primary display captured wrong screen regions due to missing virtual screen coordinate offsets.
- **Focus management failures**: `SetForegroundWindow()` returned false in multi-monitor configurations, preventing proper overlay window activation.

These issues were particularly problematic in vertical monitor arrangements (e.g., laptop screen below external monitor) where the virtual screen origin is not (0,0).

## Decision

Implement comprehensive multi-monitor support with the following changes:

### Focus Management Enhancement
- Add `AllowSetForegroundWindow()` call before `SetForegroundWindow()` to ensure permission for focus stealing.
- Add `BringWindowToTop()` as fallback when `SetForegroundWindow()` fails.
- Log all focus operations for debugging.

### Coordinate System Fixes
- Calculate region coordinates relative to virtual screen origin instead of window client area.
- Add virtual screen offset (X, Y) to selected region coordinates.
- Store virtual screen metrics globally for access in window procedures.

### Cursor Display Fixes
- Load cross cursor once at startup and cache it globally.
- Set cursor in both window class and `WM_SETCURSOR` message handler.
- Ensure cursor loading success with error logging.

### Diagnostic Logging
- Add comprehensive logging for monitor configuration, window focus, cursor operations, and coordinate calculations.
- Enable via `ENABLE_FILE_LOGGING=true` for troubleshooting.

## Consequences

### Positive

- **Full multi-monitor support**: Application works correctly on all monitor arrangements (horizontal, vertical, mixed DPI).
- **Reliable cursor display**: Crosshairs always appear during selection, regardless of monitor configuration.
- **Accurate region selection**: Captures exactly the selected area on any monitor.
- **Better debugging**: Extensive logging helps diagnose future multi-monitor issues.
- **Backward compatibility**: Single-monitor setups unaffected.

### Negative

- **Increased complexity**: Additional global state and coordinate calculations.
- **Performance overhead**: Extra logging and cursor management operations.
- **Code maintenance**: More Windows-specific code for multi-monitor edge cases.

### Neutral

- **Diagnostic capabilities**: Logging can be disabled in production builds.
- **Cross-platform impact**: Changes are Windows-only, no effect on Linux CLI.
- **Testing requirements**: Need multi-monitor test environments for validation.

## References

- **ADR-006**: DPI Awareness Implementation (related multi-monitor foundation)
- **ADR-005**: Windows GUI Subsystem (GUI architecture)
- **Issue**: Multi-monitor cursor focus and coordinate offset problems
- **Fixes**: Window focus management, virtual screen coordinate handling, cursor caching
