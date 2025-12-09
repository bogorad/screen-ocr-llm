# ADR-006: DPI Awareness Implementation

## Status

Accepted

## Date

2025-10-01

## Context

On high-DPI displays (150% scaling or higher), region selection only worked on part of the screen. Users on multi-monitor setups with different DPI scaling experienced:
- Selection overlay not covering full screen
- Mouse coordinates misaligned with visual overlay
- Partial screen capture coverage
- Inability to select regions on scaled monitors

**Root Cause:**
Windows DPI virtualization was active, causing:
- Application to receive virtualized (scaled down) coordinates
- Overlay window to cover only scaled portion of screen
- Screenshot capture to miss parts of virtual screen

## Decision

Enable DPI awareness at application startup:

**Implementation:**
```go
func enableDPIAwareness() {
    if runtime.GOOS != "windows" {
        return
    }
    
    // Prefer per-monitor DPI awareness (Win 8.1+)
    shcore := syscall.NewLazyDLL("Shcore.dll")
    setProcessDpiAwareness := shcore.NewProc("SetProcessDpiAwareness")
    const PROCESS_PER_MONITOR_DPI_AWARE = 2
    if err := setProcessDpiAwareness.Find(); err == nil {
        setProcessDpiAwareness.Call(uintptr(PROCESS_PER_MONITOR_DPI_AWARE))
        return
    }
    
    // Fallback: system DPI awareness (Vista+)
    user32 := syscall.NewLazyDLL("user32.dll")
    setProcessDPIAware := user32.NewProc("SetProcessDPIAware")
    if err := setProcessDPIAware.Find(); err == nil {
        setProcessDPIAware.Call()
    }
}

func main() {
    enableDPIAwareness()  // Call BEFORE any window/metrics usage
    // ...
}
```

**Capture Strategy:**
```go
// screenshot/screenshot.go
func Capture() ([]byte, error) {
    // Get union of all display bounds (virtual screen)
    displays, err := screenshot.GetDisplayBounds()
    unionRect := unionOfDisplays(displays)
    
    // Capture full virtual screen
    return screenshot.Capture(unionRect)
}
```

**Components Updated:**
1. `main/main.go`: Call `enableDPIAwareness()` before creating windows
2. `screenshot/screenshot.go`: Capture full virtual screen (union of displays)
3. `gui/region_selector_windows.go`: Match overlay size to virtual screen

## Consequences

### Positive

- **Full screen coverage**: Selection works across all monitors
- **Accurate coordinates**: Mouse position aligned with overlay
- **Scaled display support**: Works on 150%, 200%, 250% DPI
- **Multi-monitor support**: Different DPI scales per monitor
- **No virtualization**: Application receives true screen coordinates

### Negative

- **Windows-specific code**: DPI awareness calls only on Windows
- **Version requirements**: Per-monitor DPI requires Win 8.1+ (graceful fallback)
- **Testing complexity**: Need multi-DPI setups to test properly

### Neutral

- Performance impact negligible
- Alternative approaches (manifest) not needed
- Doesn't affect single monitor at 100% scaling

## References

- Windows API: `Shcore.SetProcessDpiAwareness` (Win 8.1+)
- Fallback: `user32.SetProcessDPIAware` (Vista+)
- DPI mode: `PROCESS_PER_MONITOR_DPI_AWARE = 2`
- Related: Virtual screen capture (union of all displays)
- User report: "150% DPI scaling, selection only worked on part of screen" → Fixed
