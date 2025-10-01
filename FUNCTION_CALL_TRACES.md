# Function Call Traces

This document traces the function call flows for three execution scenarios after the callback mechanism refactoring.

## Scenario 1: --run-once standalone (no resident)

**Entry Point:** `main/main.go:main()` with `--run-once` flag

**Call Flow:**
```
1. main() [main/main.go:42]
   ├─ flag.Parse() [line 48]
   ├─ *runOnce == true [line 51]
   ├─ singleinstance.NewClient() [line 54]
   ├─ client.TryRunOnce(ctx, stdout=false) [line 55]
   │  └─ tcpClient.TryRunOnce() [singleinstance/tcp_client.go:17]
   │     ├─ Scans port range [line 22]
   │     ├─ ping(addr, deadline) returns false (no resident) [line 24]
   │     └─ Returns (delegated=false, "", nil) [line 47]
   │
   ├─ delegated == false [line 64]
   ├─ log "No resident detected, running standalone" [line 65]
   └─ runOCROnce(stdout=false) [line 67]
      └─ runOCROnce() [main/main.go:169]
         ├─ config.Load() [line 172]
         ├─ screenshot.Init() [line 184]
         ├─ ocr.Init() [line 185]
         ├─ llm.Init() [line 186]
         ├─ clipboard.Init() [line 194]
         ├─ gui.StartRegionSelection() [line 202] ← RETURNS REGION
         │  └─ gui.StartRegionSelection() [gui/gui.go:17]
         │     ├─ StartInteractiveRegionSelection() [line 21]
         │     │  └─ StartInteractiveRegionSelection() [gui/region_selector_windows.go:38]
         │     │     ├─ Creates overlay window [lines 52-115]
         │     │     ├─ User clicks and drags
         │     │     ├─ WM_LBUTTONUP handler [line 200]
         │     │     ├─ Sends region to channel [line 224]
         │     │     └─ Returns region [line 147]
         │     └─ Returns (region, nil) [line 34]
         │
         ├─ ocr.Recognize(region) [line 211] ← PROCESSES REGION
         │  └─ ocr.Recognize() [ocr/ocr.go]
         │     ├─ screenshot.CaptureRegion(region)
         │     ├─ llm.QueryVision(imageData)
         │     └─ Returns (text, nil)
         │
         ├─ clipboard.Write(text) [line 228]
         ├─ popup.Show(text) [line 238]
         ├─ time.Sleep(3 * time.Second) [line 240]
         └─ os.Exit(0) [line 243]
```

**Key Points:**
- ✓ No callback mechanism involved
- ✓ Direct return value from `gui.StartRegionSelection()`
- ✓ OCR processing happens immediately after region selection
- ✓ Clean, linear flow

---

## Scenario 2: --run-once with active resident

**Entry Point:** `main/main.go:main()` with `--run-once` flag (resident already running)

**Call Flow:**
```
1. main() [main/main.go:42]
   ├─ flag.Parse() [line 48]
   ├─ *runOnce == true [line 51]
   ├─ singleinstance.NewClient() [line 54]
   ├─ client.TryRunOnce(ctx, stdout=false) [line 55]
   │  └─ tcpClient.TryRunOnce() [singleinstance/tcp_client.go:17]
   │     ├─ Scans port range [line 22]
   │     ├─ ping(addr, deadline) returns true (resident found!) [line 24]
   │     ├─ net.DialTimeout("tcp", addr, deadline) [line 26]
   │     ├─ Writes "CLIPBOARD\n" to connection [line 29]
   │     ├─ Waits for response [line 33]
   │     ├─ Reads "SUCCESS\n" [line 35]
   │     ├─ Reads OCR result text [line 36]
   │     └─ Returns (delegated=true, text, nil) [line 38]
   │
   ├─ delegated == true [line 61]
   ├─ log "Delegated to resident" [line 62]
   └─ return [line 63] ← CLIENT EXITS HERE

MEANWHILE, IN THE RESIDENT PROCESS:

2. eventloop.Loop.Run() [eventloop/eventloop.go:68]
   ├─ Listening on TCP port [line 75]
   ├─ Accept loop running [lines 82-91]
   ├─ conn received from srv.Next() [line 84]
   ├─ conn sent to reqCh [line 89]
   ├─ select receives conn from reqCh [line 99]
   └─ handleConn(ctx, conn) [line 103]
      └─ handleConn() [eventloop/eventloop.go:110]
         ├─ Check if busy [line 111]
         ├─ conn.Request() gets "CLIPBOARD" [line 117]
         ├─ selectRegion(ctx) [line 118] ← CALLS REGION SELECTION
         │  └─ selector.Select(ctx) [line 202]
         │     └─ windowsSelector.Select() [overlay/overlay_windows.go:16]
         │        ├─ gui.StartRegionSelection() [line 18] ← RETURNS REGION
         │        │  └─ [Same flow as Scenario 1]
         │        └─ Returns (region, false, nil) [line 28]
         │
         ├─ pool.Submit(jobCtx, region, callback) [line 134] ← SUBMITS OCR JOB
         │  └─ worker.Pool.Submit() [worker/pool.go]
         │     ├─ Spawns goroutine
         │     ├─ ocr.Recognize(region) [worker/pool.go]
         │     ├─ Calls callback(text, err) [worker/pool.go]
         │     └─ callback sends to l.results channel [line 135]
         │
         └─ Returns (job submitted)

3. eventloop.Loop.Run() [eventloop/eventloop.go:68]
   ├─ select receives result from l.results [line 104]
   └─ handleResult(res) [line 105]
      └─ handleResult() [eventloop/eventloop.go:145]
         ├─ res.conn != nil (IPC client) [line 148]
         ├─ res.err == nil [line 150]
         ├─ clipboard.Write(res.text) [line 157]
         ├─ conn.RespondSuccess(res.text) [line 158] ← SENDS TO CLIENT
         ├─ popup.Show(res.text) [line 161]
         └─ conn.Close() [line 149]
```

**Key Points:**
- ✓ Client delegates to resident via TCP
- ✓ Resident uses same `gui.StartRegionSelection()` with direct return
- ✓ OCR processing happens in worker pool
- ✓ Result sent back to client via TCP connection
- ✓ No callback mechanism involved

---

## Scenario 3: Hotkey activation with active resident

**Entry Point:** User presses hotkey (e.g., Ctrl+Alt+Q)

**Call Flow:**
```
1. hotkey.Listen() [hotkey/hotkey.go:18]
   ├─ Registered at startup [eventloop/eventloop.go:61]
   ├─ gohook event loop running [hotkey/hotkey.go:95]
   ├─ User presses hotkey combination
   ├─ All keys detected as pressed [line 124]
   ├─ log "HOTKEY COMBINATION DETECTED!" [line 108]
   ├─ callback() invoked [line 118] ← CALLBACK FROM EVENTLOOP
   │  └─ Anonymous function [eventloop/eventloop.go:61-62]
   │     └─ Sends to l.hotkeyCh channel [line 62]
   │
   └─ Returns to gohook event loop

2. eventloop.Loop.Run() [eventloop/eventloop.go:68]
   ├─ select receives from l.hotkeyCh [line 97]
   └─ handleHotkey(ctx) [line 98]
      └─ handleHotkey() [eventloop/eventloop.go:177]
         ├─ Check if busy [line 178]
         ├─ selectRegion(ctx) [line 182] ← CALLS REGION SELECTION
         │  └─ selector.Select(ctx) [line 202]
         │     └─ windowsSelector.Select() [overlay/overlay_windows.go:16]
         │        ├─ gui.StartRegionSelection() [line 18] ← RETURNS REGION
         │        │  └─ gui.StartRegionSelection() [gui/gui.go:17]
         │        │     ├─ StartInteractiveRegionSelection() [line 21]
         │        │     │  └─ [Same overlay flow as Scenario 1]
         │        │     └─ Returns (region, nil) [line 34]
         │        └─ Returns (region, false, nil) [line 28]
         │
         ├─ l.busy = true [line 191]
         ├─ pool.Submit(jobCtx, region, callback) [line 192] ← SUBMITS OCR JOB
         │  └─ worker.Pool.Submit() [worker/pool.go]
         │     ├─ Spawns goroutine
         │     ├─ ocr.Recognize(region) [worker/pool.go]
         │     ├─ Calls callback(text, err) [worker/pool.go]
         │     └─ callback sends to l.results channel [line 193]
         │
         └─ Returns

3. eventloop.Loop.Run() [eventloop/eventloop.go:68]
   ├─ select receives result from l.results [line 104]
   └─ handleResult(res) [line 105]
      └─ handleResult() [eventloop/eventloop.go:145]
         ├─ res.conn == nil (hotkey, not IPC) [line 148]
         ├─ res.err == nil [line 165]
         ├─ clipboard.Write(res.text) [line 167]
         ├─ popup.Show(res.text) [line 168]
         └─ l.busy = false [line 146]
```

**Key Points:**
- ✓ Hotkey triggers via callback to eventloop
- ✓ Eventloop uses same `gui.StartRegionSelection()` with direct return
- ✓ OCR processing happens in worker pool
- ✓ Result handled locally (no IPC connection)
- ✓ No callback mechanism for region selection

---

## Logic Verification

### All Scenarios Share Common Pattern:
1. **Region Selection**: `gui.StartRegionSelection()` returns `(screenshot.Region, error)`
2. **No Callbacks**: No callback mechanism for region selection
3. **Direct Flow**: Region → OCR → Result
4. **Worker Pool**: OCR processing uses worker pool with result callback

### Key Differences:
- **Scenario 1**: Standalone process, exits after completion
- **Scenario 2**: Client delegates to resident, resident handles everything
- **Scenario 3**: Resident handles hotkey directly

### Verification:
- ✓ All scenarios use the refactored API correctly
- ✓ No callback overwriting issues
- ✓ Clean separation of concerns
- ✓ Consistent error handling
- ✓ Proper resource cleanup

**CONCLUSION: Logic is correct and ready for testing!**

