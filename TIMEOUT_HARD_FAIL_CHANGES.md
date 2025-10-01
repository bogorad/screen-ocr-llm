# Timeout and Hard Fail Changes

## Summary

Changed timeout behavior to be a hard fail with no retries or fallbacks.

## Changes Made

### 1. Default Timeout: 15s → 10s

**File**: `eventloop/eventloop.go`

**Before**:
```go
func readDeadline() time.Duration {
    v := os.Getenv("OCR_DEADLINE_SEC")
    if v == "" {
        return 15 * time.Second  // OLD: 15 seconds
    }
    // ...
}
```

**After**:
```go
func readDeadline() time.Duration {
    v := os.Getenv("OCR_DEADLINE_SEC")
    if v == "" {
        return 10 * time.Second  // NEW: 10 seconds
    }
    // ...
}
```

**Impact**: Default OCR timeout is now 10 seconds instead of 15 seconds.

---

### 2. Removed Retry Logic

**File**: `llm/llm.go`

**Before**:
```go
const (
    openRouterURL = "https://openrouter.ai/api/v1/chat/completions"
    maxRetries    = 3              // REMOVED
    initialDelay  = 1 * time.Second // REMOVED
)

// Retry logic with exponential backoff
var lastErr error
for attempt := 0; attempt < maxRetries; attempt++ {
    if attempt > 0 {
        delay := time.Duration(float64(initialDelay) * (1.5 * float64(attempt)))
        time.Sleep(delay)
    }
    
    response, err := makeAPIRequest(request)
    if err != nil {
        log.Printf("LLM: API request failed (attempt %d/%d): %v", attempt+1, maxRetries, err)
        lastErr = err
        continue  // RETRY
    }
    // ...
}
```

**After**:
```go
const (
    openRouterURL = "https://openrouter.ai/api/v1/chat/completions"
)

// Single attempt - no retries, hard fail on any error
response, err := makeAPIRequest(request)
if err != nil {
    log.Printf("LLM: API request failed: %v", err)
    return "", fmt.Errorf("API request failed: %v", err)  // HARD FAIL
}
```

**Impact**: 
- No retries on API failure
- Immediate hard fail on any error
- Faster failure detection

---

### 3. Verified: No Provider Fallbacks

**File**: `llm/llm.go` (line 92)

```go
allowFallbacks := false  // ✅ Already set to false
prefs := &ProviderPreferences{
    Order:          config.Providers,
    AllowFallbacks: &allowFallbacks,
}
```

**Status**: ✅ Already correct - no fallbacks enabled

---

### 4. Timeout Hierarchy

The timeout enforcement happens at multiple levels:

1. **Context Timeout** (10s default): `eventloop/eventloop.go:190`
   ```go
   jobCtx, _ := context.WithTimeout(ctx, readDeadline())
   ```

2. **Worker Pool Respects Context**: `worker/pool.go:92-94`
   ```go
   case <-ctx.Done():
       // Hard fail on timeout
       return "", ctx.Err()
   ```

3. **HTTP Client Timeout** (45s): `llm/llm.go:201`
   ```go
   client := &http.Client{Timeout: 45 * time.Second}
   ```

**Priority**: Context timeout (10s) fires first → Hard fail

---

## Behavior Summary

### Before Changes

1. **Timeout**: 15 seconds default
2. **Retries**: Up to 3 attempts with exponential backoff
3. **Total Time**: Could take up to 45+ seconds (15s × 3 attempts)
4. **Fallbacks**: Disabled (already correct)

### After Changes

1. **Timeout**: 10 seconds default
2. **Retries**: None - single attempt only
3. **Total Time**: Maximum 10 seconds
4. **Fallbacks**: Disabled (unchanged)

### Failure Modes

**Network Error**:
- Before: Retry 3 times → fail after ~45s
- After: Fail immediately → fail after ~10s max

**API Error**:
- Before: Retry 3 times → fail after ~45s
- After: Fail immediately → fail after ~10s max

**Timeout**:
- Before: 15s timeout per attempt × 3 attempts
- After: 10s timeout total, hard fail

**Empty Response**:
- Before: Retry 3 times
- After: Fail immediately

---

## Configuration

Users can still override the timeout via environment variable:

```bash
# Set custom timeout (in seconds)
OCR_DEADLINE_SEC=20
```

**Default**: 10 seconds (if not set)

---

## Testing Recommendations

### Test 1: Normal Operation
- Select region with text
- Should complete within 2-5 seconds
- Text should be extracted and shown in popup

### Test 2: Timeout
- Set `OCR_DEADLINE_SEC=1` (very short)
- Select region
- Should fail with timeout error within 1 second
- Popup should show "OCR failed"

### Test 3: Network Error
- Disconnect internet
- Select region
- Should fail immediately with network error
- No retries should occur

### Test 4: API Error
- Use invalid API key
- Select region
- Should fail immediately with API error
- No retries should occur

---

## Log Output Changes

### Before (with retries)
```
LLM: API request failed (attempt 1/3): connection refused
LLM: API request failed (attempt 2/3): connection refused
LLM: API request failed (attempt 3/3): connection refused
LLM: Failed after 3 attempts: connection refused
```

### After (no retries)
```
LLM: API request failed: connection refused
```

**Result**: Cleaner logs, faster failure detection

---

## Files Modified

1. `eventloop/eventloop.go`
   - Changed default timeout: 15s → 10s
   - Updated log message

2. `llm/llm.go`
   - Removed `maxRetries` and `initialDelay` constants
   - Removed retry loop with exponential backoff
   - Single attempt with immediate hard fail
   - Simplified error messages

3. `worker/pool.go`
   - Added logging for debugging (unchanged behavior)
   - Context timeout still enforced

---

## Verification

Build successful: ✅
```bash
go build -o screen-ocr-llm.exe ./main
```

All changes applied: ✅
- Default timeout: 10s
- No retries
- No fallbacks
- Hard fail on timeout

---

## Summary

**Goal**: Make timeout a hard fail with no retries or fallbacks

**Achieved**:
- ✅ Timeout reduced to 10 seconds
- ✅ Retry logic completely removed
- ✅ Fallbacks already disabled
- ✅ Hard fail on any error
- ✅ Faster failure detection
- ✅ Cleaner logs

**Result**: OCR operations now fail fast and hard when timeout occurs, with no retry attempts or fallback mechanisms.

