# ADR-004: Popup Thread Isolation Fix

## Status

Accepted

## Date

2025-10-01

## Context

Second and subsequent hotkey activations failed with "selection cancelled" error immediately after the region selector window appeared. The issue manifested consistently on the second activation, making the application unusable for multiple captures.

**Root Cause Analysis:**
1. Popup's `WM_DESTROY` handler called `PostQuitMessage(0)`
2. This posted `WM_QUIT` to the thread's message queue
3. Region selector (running on same thread) received the stale `WM_QUIT`
4. Selection immediately cancelled on second hotkey press

**Additional Discovery:**
- Go's scheduler could reuse the popup thread for region selector
- Main goroutine was NOT locked to an OS thread
- Message queue state leaked between operations

## Decision

Implement comprehensive thread isolation:

**Phase 1: Custom Exit Message**
- Replace `PostQuitMessage` with custom `WM_EXIT_LOOP` message
- Popup message loop exits on `WM_EXIT_LOOP` instead of `WM_QUIT`
- Prevents `WM_QUIT` from leaking to other windows

**Phase 2: Thread Locking**
- Add `runtime.LockOSThread()` at start of `main()`
- Locks main goroutine to its own dedicated OS thread
- Ensures region selector always runs on different thread than popup
- Guarantees message queue isolation

**Phase 3: Message Loop Duration**
- Remove hard-coded 5-second cap in popup message loop
- Loop runs until `WM_QUIT` or `WM_EXIT_LOOP` received
- Ensures all timers process to completion
- Prevents premature loop exit during longer countdowns

**Implementation:**
```go
// main/main.go
func main() {
    runtime.LockOSThread()  // Lock to dedicated thread
    // ...
}

// notification/notification_windows.go
const WM_EXIT_LOOP = 0x0402

func (hwnd) WM_DESTROY() {
    PostMessage(hwnd, WM_EXIT_LOOP, 0, 0)  // Instead of PostQuitMessage
}

// Message loop
for {
    GetMessage(&msg, ...)
    if msg.message == WM_QUIT || msg.message == WM_EXIT_LOOP {
        break
    }
    // ...
}
```

## Consequences

### Positive

- **Reliable consecutive operations**: Second and subsequent hotkeys work perfectly
- **Clean message queue**: No stale messages leak between operations
- **Thread safety**: Each operation gets isolated message queue
- **Longer timeouts supported**: 10s+ OCR deadlines work correctly
- **Test validation**: Created `test_popup_flow.go` with 3 consecutive OCRs passing

### Negative

- **Platform-specific**: `LockOSThread()` has implications on non-Windows (but app is Windows-only)
- **Main thread dedicated**: Can't use main goroutine for other purposes
- **Debugging complexity**: Thread affinity makes debugging slightly harder

### Neutral

- Message queue behavior now explicit rather than implicit
- Performance impact negligible (thread switching already occurred)

## References

- Windows message: `WM_EXIT_LOOP = 0x0402`
- Go runtime: `runtime.LockOSThread()`
- Test: `test_popup_flow.go` (3 consecutive operations)
- Related: Countdown popup implementation (ADR-005 implied)
- Log evidence: Message queue properly flushed (0x402 = WM_EXIT_LOOP)
