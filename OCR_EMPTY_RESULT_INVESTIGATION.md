# OCR Empty Result Investigation

## Issue Report
User started resident, selected screen rectangle, OCR returned empty result.

## Log Analysis

### What Worked ✅
1. **Hotkey Detection**: Ctrl+Win+E detected correctly (line 24)
2. **Region Selection**: User selected region (0,0) to (1107,226) - 1107x226 pixels (line 176)
3. **Screenshot Capture**: Region captured successfully (line 177)
4. **Provider Configuration**: Using 3 providers: crusoe/bf16, novita/bf16, deepinfra/bf16 (line 178)
5. **API Request**: Request sent with provider preferences (line 179)
6. **Popup Shown**: Notification popup displayed (lines 180-189)

### What's Missing ❌
1. **No API Response Logging**: No log entry showing what the API returned
2. **No Text Extraction Logging**: No log showing how many characters were extracted
3. **No Error Logging**: If API failed or returned empty, no error was logged

### Log Gap
```
Line 179: LLM: API request includes provider preferences: ...
Line 180: Popup: Starting single popup thread
```

**Between lines 179-180, the entire API call, response parsing, and text extraction happened with NO LOGGING.**

## Root Cause

The OCR pipeline completed but there was insufficient logging to diagnose what happened:
- Did the API return an error?
- Did the API return empty text?
- Did the API return "NO_TEXT_FOUND"?
- Was there a network error?
- Did the retry logic kick in?

**We cannot tell from the existing logs.**

## Logging Added

### 1. API Request Failure Logging
```go
if err != nil {
    log.Printf("LLM: API request failed (attempt %d/%d): %v", attempt+1, maxRetries, err)
    lastErr = err
    continue
}
```

### 2. Empty Response Logging
```go
if len(response.Choices) == 0 {
    log.Printf("LLM: API response has no choices (attempt %d/%d)", attempt+1, maxRetries)
    lastErr = fmt.Errorf("no choices in API response")
    continue
}
```

### 3. Text Extraction Logging
```go
extractedText := response.Choices[0].Message.Content
log.Printf("LLM: API returned text: %d characters", len(extractedText))
if extractedText == "" || extractedText == "NO_TEXT_FOUND" {
    log.Printf("LLM: No text detected in image (response was: %q)", extractedText)
    return "", fmt.Errorf("no text detected in image")
}
```

### 4. Success Logging
```go
extractedText = cleanExtractedText(extractedText)
log.Printf("LLM: Successfully extracted %d characters", len(extractedText))
return extractedText, nil
```

### 5. Final Failure Logging
```go
log.Printf("LLM: Failed after %d attempts: %v", maxRetries, lastErr)
return "", fmt.Errorf("failed after %d attempts: %v", maxRetries, lastErr)
```

### 6. HTTP Response Status Logging
```go
log.Printf("LLM: API response status: %d %s", resp.StatusCode, resp.Status)
```

### 7. API Error Logging
```go
if response.Error != nil {
    log.Printf("LLM: API error response: %s (type: %s, code: %v)", response.Error.Message, response.Error.Type, response.Error.Code)
    return nil, fmt.Errorf("API error: %s (type: %s, code: %v)", response.Error.Message, response.Error.Type, response.Error.Code)
}
```

### 8. Response Parsing Success Logging
```go
log.Printf("LLM: API response parsed successfully, %d choices", len(response.Choices))
return &response, nil
```

## Expected Log Output (After Fix)

### Success Case
```
LLM: Using provider preferences: order=[crusoe/bf16 novita/bf16 deepinfra/bf16], allow_fallbacks=false
LLM: API request includes provider preferences: ...
LLM: API response status: 200 OK
LLM: API response parsed successfully, 1 choices
LLM: API returned text: 2198 characters
LLM: Successfully extracted 2198 characters
```

### Empty Text Case
```
LLM: Using provider preferences: order=[crusoe/bf16 novita/bf16 deepinfra/bf16], allow_fallbacks=false
LLM: API request includes provider preferences: ...
LLM: API response status: 200 OK
LLM: API response parsed successfully, 1 choices
LLM: API returned text: 0 characters
LLM: No text detected in image (response was: "")
```

### API Error Case
```
LLM: Using provider preferences: order=[crusoe/bf16 novita/bf16 deepinfra/bf16], allow_fallbacks=false
LLM: API request includes provider preferences: ...
LLM: API response status: 400 Bad Request
LLM: API error response: Invalid request (type: invalid_request_error, code: 400)
LLM: API request failed (attempt 1/3): API error: Invalid request (type: invalid_request_error, code: 400)
```

### Network Error Case
```
LLM: Using provider preferences: order=[crusoe/bf16 novita/bf16 deepinfra/bf16], allow_fallbacks=false
LLM: API request includes provider preferences: ...
LLM: API request failed (attempt 1/3): API request failed: dial tcp: lookup openrouter.ai: no such host
LLM: API request failed (attempt 2/3): API request failed: dial tcp: lookup openrouter.ai: no such host
LLM: API request failed (attempt 3/3): API request failed: dial tcp: lookup openrouter.ai: no such host
LLM: Failed after 3 attempts: API request failed: dial tcp: lookup openrouter.ai: no such host
```

## Next Steps

1. **Rebuild the program**: `go build -o screen-ocr-llm.exe ./main`
2. **Run the resident again**: `.\screen-ocr-llm.exe`
3. **Trigger OCR**: Press Ctrl+Win+E and select a region
4. **Check the log**: Look for the new log entries to diagnose the issue

## Possible Causes of Empty Result

Based on the new logging, we'll be able to identify:

1. **API returned empty text**: Model couldn't extract text from the image
   - Possible reasons: Image too small, low quality, no text in region
   - Solution: Select a larger region with clearer text

2. **API returned "NO_TEXT_FOUND"**: Model explicitly said no text
   - Possible reasons: Selected region has no text
   - Solution: Select a region with actual text

3. **API error**: OpenRouter or provider returned an error
   - Possible reasons: Invalid API key, rate limit, provider unavailable
   - Solution: Check API key, wait and retry, or change providers

4. **Network error**: Couldn't reach OpenRouter
   - Possible reasons: No internet, firewall blocking, DNS issues
   - Solution: Check internet connection, firewall settings

5. **Response parsing error**: Couldn't parse API response
   - Possible reasons: Unexpected response format, corrupted data
   - Solution: Check OpenRouter API status, update code if API changed

## Testing Recommendations

### Test 1: Known Good Image
Select a region with clear, large text (e.g., a heading or title) to verify the pipeline works.

### Test 2: Small Region
Select a very small region (< 50x50 pixels) to see if size matters.

### Test 3: No Text Region
Select a region with no text (e.g., blank space, image only) to see the "NO_TEXT_FOUND" response.

### Test 4: Network Test
Run `go run test_ocr_with_image.go` to test the API with test-image.png and verify network connectivity.

## Files Modified

- `llm/llm.go`: Added comprehensive logging throughout the OCR pipeline
  - Lines 160-162: API request failure logging
  - Lines 167-169: Empty response logging
  - Lines 172-176: Text extraction and empty text logging
  - Lines 181-182: Success logging
  - Lines 185-186: Final failure logging
  - Line 223: HTTP response status logging
  - Lines 231-233: API error logging
  - Line 241: Response parsing success logging

## Summary

The issue was **insufficient logging** in the OCR pipeline. The API call completed but we couldn't tell what happened because there were no log entries between the request and the popup.

With the new logging, we'll be able to see:
- HTTP response status
- API errors
- Empty responses
- Text extraction results
- Retry attempts
- Final success or failure

**Next action**: Rebuild and test again to see the detailed logs and identify the actual cause of the empty result.

