# Countdown Popup Implementation

## Summary

Implemented a countdown popup system that shows OCR progress in real-time and seamlessly transitions to showing results.

## Features

1. **Immediate Feedback**: Popup appears instantly when OCR starts
2. **Live Countdown**: Updates every second showing time remaining
3. **Seamless Transition**: Replaces countdown with result text when OCR completes
4. **Silent Timeout**: Closes silently if OCR times out
5. **User Interaction**: Click anywhere to close popup immediately

## Implementation Details

### Windows API Additions

**New Procs**:
- `procInvalidateRect`: Forces window repaint
- `procPostMessage`: Sends messages to window from other threads

**New Constants**:
- `WM_USER = 0x0400`: Base for custom messages
- `WM_UPDATE_TEXT = WM_USER + 1`: Custom message for text updates
- `TIMER_CLOSE = 1`: Timer ID for 3-second close
- `TIMER_COUNTDOWN = 2`: Timer ID for 1-second countdown

### State Management

**Global Variables**:
```go
currentPopupHwnd syscall.Handle  // Handle to current popup window
isCountdownMode bool              // Whether showing countdown or result
countdownRemaining int            // Seconds remaining in countdown
```

### Window Procedure Changes

**WM_TIMER Handler**:
- `TIMER_COUNTDOWN`: Decrements counter, updates text, closes at zero
- `TIMER_CLOSE`: Closes popup after 3 seconds (result display)

**WM_UPDATE_TEXT Handler**:
- Stops countdown timer
- Switches to result mode
- Sets 3-second close timer
- Forces repaint with new text

**WM_DESTROY Handler**:
- Clears `currentPopupHwnd`
- Resets `isCountdownMode`

### Public API

**popup.StartCountdown(timeoutSeconds int)**:
- Creates popup with initial countdown text
- Starts 1-second timer for updates
- Returns immediately (non-blocking)

**popup.UpdateText(text string)**:
- Updates popup text
- Switches from countdown to result mode
- Shows result for 3 seconds
- Returns immediately (non-blocking)

**popup.Close()**:
- Closes current popup if any
- Returns immediately (non-blocking)

### Integration with OCR Flow

**eventloop.handleHotkey()**:
```go
// After region selection
deadline := readDeadline()
timeoutSeconds := int(deadline.Seconds())
_ = popup.StartCountdown(timeoutSeconds)  // Start countdown immediately

// Submit OCR job
l.pool.Submit(jobCtx, region, callback)
```

**eventloop.handleResult()**:
```go
// On success
_ = popup.UpdateText(res.text)  // Replace countdown with result

// On error/timeout
_ = popup.Close()  // Close silently
```

## Behavior

### Success Case

1. User presses hotkey (Ctrl+Win+E)
2. User selects region
3. **Popup appears**: "OCR in progress... 10 seconds remaining"
4. **Every second**: "OCR in progress... 9 seconds remaining"
5. **Every second**: "OCR in progress... 8 seconds remaining"
6. ... (continues updating)
7. **OCR completes** (e.g., after 3 seconds)
8. **Popup updates**: Shows extracted text
9. **After 3 seconds**: Popup closes automatically

### Timeout Case

1. User presses hotkey
2. User selects region
3. **Popup appears**: "OCR in progress... 10 seconds remaining"
4. **Every second**: Countdown decrements
5. **Countdown reaches 0**: Popup closes silently
6. **No error message shown**

### Error Case

1. User presses hotkey
2. User selects region
3. **Popup appears**: "OCR in progress... 10 seconds remaining"
4. **API error occurs** (e.g., network failure)
5. **Popup closes immediately**
6. **No error message shown** (silent fail per requirements)

## Files Modified

### notification/notification_windows.go
- Added `procInvalidateRect` and `procPostMessage`
- Added `WM_UPDATE_TEXT`, `TIMER_CLOSE`, `TIMER_COUNTDOWN` constants
- Added state variables: `currentPopupHwnd`, `isCountdownMode`, `countdownRemaining`
- Modified `wndProc` to handle countdown timer and text updates
- Modified `createAndShowPopup` to support countdown mode
- Added `StartCountdownPopup()` function
- Added `UpdatePopupText()` function
- Added `ClosePopup()` function

### popup/popup.go
- Added `StartCountdown(timeoutSeconds int)` function
- Added `UpdateText(text string)` function
- Added `Close()` function

### eventloop/eventloop.go
- Modified `handleHotkey()` to start countdown popup
- Modified `handleResult()` to update or close popup based on outcome

## Testing Checklist

### Test 1: Normal Operation
- [ ] Start resident
- [ ] Press Ctrl+Win+E
- [ ] Select region with text
- [ ] Verify countdown appears immediately
- [ ] Verify countdown updates every second
- [ ] Verify countdown shows correct remaining time
- [ ] Verify popup updates with result text
- [ ] Verify popup closes after 3 seconds

### Test 2: Fast OCR (< 1 second)
- [ ] Select small region
- [ ] Verify countdown shows "10 seconds remaining"
- [ ] Verify popup updates to result before countdown changes
- [ ] Verify result shows for 3 seconds

### Test 3: Timeout
- [ ] Set `OCR_DEADLINE_SEC=3`
- [ ] Select region
- [ ] Verify countdown shows "3 seconds remaining"
- [ ] Verify countdown updates: 3 → 2 → 1 → 0
- [ ] Verify popup closes at 0
- [ ] Verify no error message shown

### Test 4: Click to Close
- [ ] Start OCR
- [ ] Click on countdown popup
- [ ] Verify popup closes immediately
- [ ] Verify OCR continues in background

### Test 5: Multiple Rapid Triggers
- [ ] Press hotkey
- [ ] Select region
- [ ] Immediately press hotkey again
- [ ] Verify "Busy, please retry" message
- [ ] Verify countdown popup remains

## Technical Notes

### Thread Safety
- All popup state access protected by `currentPopupMutex`
- `PostMessage` is thread-safe (Windows API guarantee)
- Popup window runs on dedicated thread (locked to OS thread)

### Timer Precision
- Windows timer resolution: ~15ms
- 1-second countdown timer is accurate enough for user feedback
- Actual OCR time may vary, countdown is independent

### Memory Management
- Window handle cleared on `WM_DESTROY`
- Timers killed before window destruction
- No memory leaks

### Edge Cases Handled
- Popup already exists: Destroyed before creating new one
- Update text when no popup: Logged and ignored
- Close when no popup: Ignored
- Countdown reaches zero: Popup closes silently

## Performance Impact

- **Minimal**: One additional timer (1-second interval)
- **No blocking**: All operations are asynchronous
- **No extra threads**: Uses existing popup thread
- **Low CPU**: Timer only fires once per second

## Future Enhancements

1. **Progress Bar**: Add visual progress bar alongside countdown
2. **Animated Icon**: Show spinning icon during OCR
3. **Configurable Position**: Allow user to set popup position
4. **Sound Notification**: Optional sound on completion
5. **Fade Animation**: Smooth fade in/out transitions

## Summary

The countdown popup implementation provides real-time feedback to users during OCR operations, making the wait time visible and managing expectations. The seamless transition from countdown to results creates a polished user experience, while silent timeout handling avoids unnecessary error messages.

**Key Benefits**:
- ✅ Immediate visual feedback
- ✅ Clear time expectations
- ✅ Seamless result display
- ✅ Silent failure handling
- ✅ Non-blocking implementation
- ✅ Thread-safe
- ✅ Low performance impact

