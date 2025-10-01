# Manual Testing Guide

This guide provides step-by-step instructions for manually testing the three execution scenarios after the callback mechanism refactoring.

## Prerequisites

1. Build the program: `go build -o screen-ocr-llm.exe ./main`
2. Ensure `.env` file is configured with valid API key
3. Have test-image.png open in an image viewer for region selection

## Automated Test (Baseline)

**Purpose**: Verify the OCR pipeline works correctly with test-image.png

**Steps**:
```bash
go run test_ocr_with_image.go
```

**Expected Result**:
- ✓ Loads test-image.png (206,086 bytes)
- ✓ Performs OCR successfully
- ✓ Extracts ~2,198 characters
- ✓ Displays coherent text about Trump administration tariffs

**Status**: ✅ PASSED (verified 2025-01-03)

---

## Scenario 1: --run-once standalone (no resident)

**Purpose**: Test standalone execution without a resident process

**Setup**:
1. Ensure no resident process is running (check Task Manager for screen-ocr-llm.exe)
2. Open test-image.png in an image viewer (e.g., Windows Photos)

**Steps**:
```bash
.\screen-ocr-llm.exe --run-once
```

**Expected Behavior**:
1. Log message: "No resident detected, running standalone"
2. Overlay window appears (transparent with crosshair cursor)
3. Click and drag to select a region of test-image.png
4. Release mouse button
5. Overlay disappears
6. OCR processing begins
7. Result copied to clipboard
8. Popup notification appears showing extracted text
9. Program exits after 3 seconds

**Verification**:
- [ ] Overlay appeared correctly
- [ ] Click-drag worked smoothly
- [ ] Mouse release triggered OCR processing
- [ ] Text copied to clipboard (paste into notepad to verify)
- [ ] Popup showed extracted text
- [ ] Program exited cleanly

**Function Call Trace**: See FUNCTION_CALL_TRACES.md → Scenario 1

**Key Points to Verify**:
- ✓ No callback overwriting issues
- ✓ Direct return value from `gui.StartRegionSelection()`
- ✓ OCR processing happens immediately after region selection
- ✓ Clean, linear flow

---

## Scenario 2: --run-once with active resident

**Purpose**: Test delegation from client to resident process

**Setup**:
1. Start resident process in one terminal:
   ```bash
   .\screen-ocr-llm.exe
   ```
2. Wait for log message: "Resident listening on 127.0.0.1:XXXXX"
3. Open test-image.png in an image viewer
4. Open a second terminal for the client

**Steps** (in second terminal):
```bash
.\screen-ocr-llm.exe --run-once
```

**Expected Behavior**:

**Client Process**:
1. Log message: "Delegated to resident"
2. Client exits immediately

**Resident Process**:
1. Receives IPC request
2. Overlay window appears
3. Click and drag to select a region of test-image.png
4. Release mouse button
5. Overlay disappears
6. OCR processing begins in worker pool
7. Result copied to clipboard
8. Popup notification appears
9. Resident continues running (does NOT exit)

**Verification**:
- [ ] Client detected resident and delegated
- [ ] Client exited immediately
- [ ] Resident handled the request
- [ ] Overlay appeared correctly
- [ ] Click-drag worked smoothly
- [ ] Mouse release triggered OCR processing
- [ ] Text copied to clipboard
- [ ] Popup showed extracted text
- [ ] Resident still running after completion

**Function Call Trace**: See FUNCTION_CALL_TRACES.md → Scenario 2

**Key Points to Verify**:
- ✓ Client delegates to resident via TCP
- ✓ Resident uses same `gui.StartRegionSelection()` with direct return
- ✓ OCR processing happens in worker pool
- ✓ Result sent back to client via TCP connection
- ✓ No callback mechanism involved

---

## Scenario 3: Hotkey activation with active resident

**Purpose**: Test hotkey-triggered OCR with resident process

**Setup**:
1. Start resident process:
   ```bash
   .\screen-ocr-llm.exe
   ```
2. Wait for log message: "Resident listening on 127.0.0.1:XXXXX"
3. Open test-image.png in an image viewer
4. Check system tray for the application icon

**Steps**:
1. Press the configured hotkey (default: Ctrl+Alt+Q)

**Expected Behavior**:
1. Log message: "HOTKEY COMBINATION DETECTED!"
2. Overlay window appears immediately
3. Click and drag to select a region of test-image.png
4. Release mouse button
5. Overlay disappears
6. OCR processing begins in worker pool
7. Result copied to clipboard
8. Popup notification appears
9. Resident continues running

**Verification**:
- [ ] Hotkey triggered correctly
- [ ] Overlay appeared immediately
- [ ] Click-drag worked smoothly
- [ ] Mouse release triggered OCR processing
- [ ] Text copied to clipboard
- [ ] Popup showed extracted text
- [ ] Resident still running after completion
- [ ] Can trigger hotkey again for another OCR

**Function Call Trace**: See FUNCTION_CALL_TRACES.md → Scenario 3

**Key Points to Verify**:
- ✓ Hotkey triggers via callback to eventloop
- ✓ Eventloop uses same `gui.StartRegionSelection()` with direct return
- ✓ OCR processing happens in worker pool
- ✓ Result handled locally (no IPC connection)
- ✓ No callback mechanism for region selection

---

## Common Issues and Troubleshooting

### Issue: Overlay doesn't appear
**Possible Causes**:
- Another instance is already running (check Task Manager)
- Display scaling issues (try on primary monitor)
- Windows permissions (run as administrator)

**Solution**: Kill all instances and restart

### Issue: Click-drag doesn't work
**Possible Causes**:
- Mouse capture not working
- Overlay window not receiving events
- Windows focus issues

**Solution**: Check logs for WM_LBUTTONDOWN and WM_LBUTTONUP messages

### Issue: OCR doesn't process after mouse release
**Possible Causes**:
- Region too small (minimum 5x5 pixels)
- Context cancelled
- Worker pool busy

**Solution**: Check logs for "Processing region" message

### Issue: No text copied to clipboard
**Possible Causes**:
- Clipboard initialization failed
- Another application has clipboard lock
- OCR failed

**Solution**: Check logs for "OCR failed" or "Failed to write to clipboard"

---

## Test Results Template

Copy this template and fill in results:

```
## Test Results - [Date]

### Automated Test
- Status: [ ] PASS / [ ] FAIL
- Notes: 

### Scenario 1: --run-once standalone
- Status: [ ] PASS / [ ] FAIL
- Overlay: [ ] OK / [ ] ISSUE
- Click-drag: [ ] OK / [ ] ISSUE
- OCR: [ ] OK / [ ] ISSUE
- Clipboard: [ ] OK / [ ] ISSUE
- Popup: [ ] OK / [ ] ISSUE
- Notes:

### Scenario 2: --run-once with resident
- Status: [ ] PASS / [ ] FAIL
- Delegation: [ ] OK / [ ] ISSUE
- Overlay: [ ] OK / [ ] ISSUE
- Click-drag: [ ] OK / [ ] ISSUE
- OCR: [ ] OK / [ ] ISSUE
- Clipboard: [ ] OK / [ ] ISSUE
- Popup: [ ] OK / [ ] ISSUE
- Notes:

### Scenario 3: Hotkey with resident
- Status: [ ] PASS / [ ] FAIL
- Hotkey: [ ] OK / [ ] ISSUE
- Overlay: [ ] OK / [ ] ISSUE
- Click-drag: [ ] OK / [ ] ISSUE
- OCR: [ ] OK / [ ] ISSUE
- Clipboard: [ ] OK / [ ] ISSUE
- Popup: [ ] OK / [ ] ISSUE
- Notes:

### Overall Assessment
- All scenarios working: [ ] YES / [ ] NO
- Callback refactoring successful: [ ] YES / [ ] NO
- Ready for production: [ ] YES / [ ] NO
```

---

## Next Steps

After completing manual testing:
1. Fill in the test results template
2. Update STATUS.md with results
3. If all tests pass, mark the mission as complete
4. If any tests fail, document issues and create fix plan

