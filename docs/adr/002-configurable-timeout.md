# ADR-002: Configurable Timeout Implementation

## Status

Accepted

## Date

2025-11-10

## Context

OCR timeout was hardcoded to 15 seconds in `eventloop/eventloop.go`. Users needed the ability to configure this based on their network conditions, LLM provider speed, and image complexity. A junior developer attempted to implement configurable timeout via `OCR_DEADLINE_SEC` environment variable but accidentally deleted critical functions during refactoring.

**Issues Found:**
- Deleted `handleResult()` function (processes OCR results)
- Deleted `handleHotkey()` function (handles hotkey activation)
- Broke `handleConn()` callback (passed `conn: nil` instead of actual connection)
- Context leaks from unused cancel functions
- Inconsistent timeout default (15s vs 20s)

## Decision

Implement configurable timeout with proper architecture:

1. **Add configuration field**: `config.OCRDeadlineSec int`
2. **Update eventloop**: 
   - `New()` accepts `*config.Config` parameter
   - Store `deadline time.Duration` in `Loop` struct
   - Default to 20 seconds if not configured
3. **Use deadline consistently**:
   - Hotkey path: `handleHotkey()` uses `l.deadline`
   - Delegation path: `handleConn()` uses `l.deadline`
   - Standalone path: `runOCROnce()` uses `cfg.OCRDeadlineSec`
4. **Restore deleted functions**:
   - Restore complete `handleResult()` implementation
   - Restore complete `handleHotkey()` implementation
   - Fix `handleConn()` to pass connection properly
5. **Fix context leaks**: Add `defer cancel()` to all timeout contexts

**Configuration:**
```go
// config/config.go
type Config struct {
    // ...
    OCRDeadlineSec int  // Default: 20
}

// eventloop/eventloop.go
func New(cfg *config.Config) *Loop {
    deadlineSec := 20
    if cfg != nil && cfg.OCRDeadlineSec > 0 {
        deadlineSec = cfg.OCRDeadlineSec
    }
    return &Loop{
        deadline: time.Duration(deadlineSec) * time.Second,
        // ...
    }
}
```

## Consequences

### Positive

- **User control**: Users can adjust timeout for their environment
- **Better defaults**: 20s default (up from 15s) accommodates slower networks
- **Consistent behavior**: Same timeout used across all code paths
- **No context leaks**: Proper cleanup with defer cancel()
- **Architecture preserved**: Critical functions restored correctly

### Negative

- **Configuration complexity**: One more setting for users to understand
- **Junior dev confusion**: Shows need for better code review process

### Neutral

- `.env.example` updated with `OCR_DEADLINE_SEC=20`
- All imports updated to use `screen-ocr-llm/src/config`

## References

- Environment variable: `OCR_DEADLINE_SEC` (integer, seconds)
- Default value: 20 seconds
- Verified working: OCR test extracted 2,363 characters from test-image.png
- Related: Removed retry logic remains a single-attempt behavior in the OCR request flow
