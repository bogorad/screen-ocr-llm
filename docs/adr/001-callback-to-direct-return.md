# ADR-001: Callback to Direct Return Refactoring

## Status

Accepted

## Date

2025-10-01

## Context

The region selection mechanism used a callback pattern where `gui.SetRegionSelectionCallback()` was called twice - once in `main/main.go` to set the OCR processing callback, and again in `overlay/overlay_windows.go` to set a synchronous adapter. This caused the OCR callback to be overwritten, preventing OCR processing from executing after region selection.

**Evidence:**
- Region selection completed successfully
- Window destruction occurred
- OCR callback never invoked (overwritten)
- User reported "mouse click-drag starts but doesn't trigger OCR"

## Decision

Refactor the callback mechanism to use direct return values:

1. Change `gui.StartRegionSelection()` from `error` to `(screenshot.Region, error)`
2. Remove callback mechanism entirely:
   - Delete `gui.SetRegionSelectionCallback()`
   - Delete `gui.RegionSelectionCallback` type
   - Delete `gui.regionCallback` variable
3. Update `overlay/overlay_windows.go` to return region directly
4. Move OCR processing to eventloop, after `Select()` returns

**Architecture Flow (After Fix):**
```
Hotkey pressed
  → eventloop.handleHotkey()
    → overlay.windowsSelector.Select()
      → gui.StartRegionSelection()
        → StartInteractiveRegionSelection() [blocks until selection]
        → Returns region
      ← Returns region
    ← Returns region
    → eventloop processes OCR on returned region
```

## Consequences

### Positive

- **No callback overwriting**: Impossible to overwrite since callbacks don't exist
- **Simpler flow**: Direct return values are easier to understand than callbacks
- **Centralized OCR**: Processing logic in one place (eventloop)
- **Easier to test**: Synchronous calls are simpler to test than async callbacks
- **Clearer ownership**: Region selector owns region, eventloop owns OCR

### Negative

- **Breaking change**: All callers of `StartRegionSelection()` must be updated
- **Blocking calls**: Region selection blocks until complete (but this was already true)

### Neutral

- Code organization changes but overall complexity similar
- Number of function calls remains the same

## References

- Fixed all test files: `gui_test.go`, `validation_test.go`, `debug_test.go`
- Updated `main/main.go` runonce mode
- Cleaned up `hotkey/hotkey.go`
- Created automated test with test-image.png (2,198 characters extracted)
