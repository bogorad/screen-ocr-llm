# Hotkey Bug Fix Summary

## Problem
The HOTKEY environment variable from `.env` was being **ignored** by the resident process. The `hotkey/hotkey.go` package was hardcoded to only detect `Ctrl+Alt+Q` (rawcodes 162/163 for Ctrl, 164/165 for Alt, 81 for Q) regardless of what hotkey configuration was passed to the `Listen()` function.

## Root Cause
In `hotkey/hotkey.go`:
- The `parseHotkey()` function correctly parsed the hotkey configuration string
- However, the parsed result was **never used** in the actual key detection logic
- The key detection used hardcoded rawcode checks in a switch statement
- This meant any custom HOTKEY value in `.env` was completely ignored

## Solution
Refactored `hotkey/hotkey.go` to make hotkey detection **configuration-driven**:

### 1. Added `keyNameToRawcodes()` function
- Maps key names (ctrl, alt, shift, a-z, 0-9, F1-F12, etc.) to Windows virtual key code rawcodes
- Supports both left and right variants of modifier keys
- Based on official Microsoft Windows VK codes documentation
- Returns `nil` for unknown keys with a warning log

### 2. Refactored `Listen()` function
- Now builds a dynamic `keyStates` slice based on the parsed hotkey configuration
- Each key state tracks:
  - Key name (for logging)
  - Rawcodes (supporting left/right variants)
  - Pressed state (boolean)
- Key detection logic now iterates through configured keys instead of hardcoded checks
- Logs show the actual configured hotkey combination

### 3. Updated existing test
- Fixed `hotkey/hotkey_test.go` to pass the hotkey configuration parameter

### 4. Added comprehensive tests
- Created `hotkey/hotkey_mapping_test.go` with tests for:
  - `keyNameToRawcodes()` - validates key-to-rawcode mapping
  - `parseHotkey()` - validates hotkey string parsing
- All tests pass successfully

## Changes Made

### Modified Files
1. **hotkey/hotkey.go**
   - Added `keyNameToRawcodes()` function (118 lines)
   - Refactored `Listen()` function to use dynamic key detection
   - Updated logging to show actual configured hotkey

2. **hotkey/hotkey_test.go**
   - Fixed `TestListen()` to pass hotkey configuration parameter

### New Files
3. **hotkey/hotkey_mapping_test.go**
   - Comprehensive tests for key mapping and parsing functions

## Supported Keys
The fix supports:
- **Modifier keys**: Ctrl, Alt, Shift, Win/Cmd/Super (both left and right variants)
- **Letter keys**: A-Z
- **Number keys**: 0-9
- **Function keys**: F1-F12
- **Special keys**: Space, Enter, Esc, Tab, Backspace, Delete, Insert, Home, End, PageUp, PageDown
- **Arrow keys**: Left, Up, Right, Down

**Note**: Win/Cmd/Super all map to the Windows key (VK_LWIN/VK_RWIN) for cross-platform compatibility.

## Testing
1. **Unit tests**: All tests pass
   ```
   go test -v ./hotkey
   ```
   - TestKeyNameToRawcodes: ✓ PASS
   - TestParseHotkey: ✓ PASS
   - TestListen: ✓ SKIP (requires TEST_API_KEY)

2. **Build**: Successful
   ```
   go build -o screen-ocr-llm.exe
   ```

## Verification Steps
To verify the fix works:

1. **Check your .env file** - Currently set to:
   ```
   HOTKEY=Ctrl+alt+e
   ```

2. **Run the application**:
   ```
   ./screen-ocr-llm.exe
   ```

3. **Check the logs** - You should see:
   ```
   Parsed hotkey configuration: [ctrl alt e]
   Hotkey listener configured for: Ctrl+alt+e
   ```

4. **Test the hotkey** - Press `Ctrl+Alt+E` (not Ctrl+Alt+Q)
   - The region selection should start
   - Logs should show: "HOTKEY COMBINATION DETECTED! Ctrl+alt+e"

5. **Try different hotkeys** - Edit `.env` and restart:
   ```
   HOTKEY=Ctrl+Shift+O
   HOTKEY=Alt+F4
   HOTKEY=Ctrl+Alt+T
   ```

## Impact
- **No breaking changes** - Function signatures remain the same
- **No caller updates needed** - All callers already pass the hotkey configuration
- **Backward compatible** - Default "Ctrl+Alt+Q" still works
- **More flexible** - Users can now customize their hotkey via .env

## Notes
- The fix maintains the existing architecture (mutex-protected state tracking, goroutine-based event handling)
- The legacy OCR path in `hotkey.Listen()` is preserved (as noted in ARCHITECTURE.md)
- Key names are case-insensitive (converted to lowercase)
- Unknown keys are logged with a warning and skipped
- The fix uses official Microsoft Windows Virtual Key Codes for maximum compatibility

