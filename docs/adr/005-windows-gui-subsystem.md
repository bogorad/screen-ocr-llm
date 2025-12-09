# ADR-005: Windows GUI Subsystem

## Status

Accepted

## Date

2025-10-01

## Context

When running the Windows executable, a console window appeared alongside the application, which was:
- Unprofessional for a GUI application
- Confusing for end users
- Unnecessary since the app is tray-based

The default Go build creates a console application that opens a terminal window. This is appropriate for CLI tools but not for GUI applications that should run silently in the background.

**User Experience Issues:**
- Console window visible behind tray app
- Users might accidentally close console, terminating app
- Log output to console is invisible in GUI builds anyway
- Non-standard for Windows tray applications

## Decision

Build as Windows GUI subsystem application for all user-facing modes:

**Build Flag:**
```bash
go build -ldflags "-H=windowsgui" -o screen-ocr-llm.exe ./src/main
```

**Changes:**
1. **build.cmd**: Use GUI subsystem flag
2. **Makefile**: 
   - Default `build` target uses GUI subsystem when `OS=Windows_NT`
   - `build-windows` remains explicit GUI build
3. **Documentation**: Update README.md and BUILD_INSTRUCTIONS.md

**Modes Affected:**
- ✅ Resident mode: No console (tray icon only)
- ✅ --run-once standalone: No console (selection UI → popup → exit)
- ✅ --run-once delegated: No console (client exits, resident shows UI)

**Logging Strategy:**
- `ENABLE_FILE_LOGGING=true` in `.env` for diagnostics
- Logs written to `screen_ocr_debug.log` (size-rotated)
- In GUI builds, stdout/stderr are hidden (by design)

## Consequences

### Positive

- **Professional appearance**: No console window
- **Tray-only UI**: Clean system tray experience
- **Standard behavior**: Matches other Windows tray applications
- **User confusion eliminated**: Can't accidentally close console
- **Silent operation**: Background service as intended

### Negative

- **No stdout/stderr**: Console output invisible (must use file logging)
- **Debugging harder**: Can't see immediate output without enabling logging
- **Testing complexity**: Some tests may need console output

### Neutral

- CI/tests unaffected (`go test` still uses console)
- Non-Windows builds unchanged
- No runtime code changes (build-time only)
- File logging becomes essential for diagnostics

## References

- Windows subsystem flag: `-ldflags "-H=windowsgui"`
- Alternative investigated: Windows manifest (not needed)
- Logging controlled by: `ENABLE_FILE_LOGGING` in `.env`
- Log file: `screen_ocr_debug.log` (size-rotated)
- Related: Startup LLM ping shows blocking error dialog on failure
