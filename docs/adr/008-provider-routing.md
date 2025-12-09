# ADR-008: Provider Routing Support

## Status

Accepted

## Date

2025-10-01

## Context

OpenRouter supports multiple LLM providers for each model. Users need control over which providers are used because:
- **Performance varies**: Different providers have different latency
- **Availability varies**: Providers may be down or rate-limited
- **Cost varies**: Some providers are cheaper than others
- **Quality varies**: Providers may use different quantizations (bf16, fp8, int8)

**User Request:**
"PROVIDERS= setting is ignored"

**Investigation Revealed:**
- Production code was passing `Providers` correctly
- Test files were missing `Providers` field
- No logging to verify provider usage
- Users couldn't tell if routing was working

## Decision

Implement explicit provider routing with comprehensive logging:

**Configuration:**
```bash
# .env
MODEL=google/gemma-3-12b-it
PROVIDERS=crusoe/bf16,novita/bf16,deepinfra/bf16
```

**OpenRouter API Request:**
```json
{
  "model": "google/gemma-3-12b-it",
  "provider": {
    "order": ["crusoe/bf16", "novita/bf16", "deepinfra/bf16"],
    "allow_fallbacks": false
  }
}
```

**Implementation:**
```go
// config/config.go
type Config struct {
    Providers []string  // Comma-separated in env, array in code
    // ...
}

func Load() (*Config, error) {
    var providers []string
    if providersStr := os.Getenv("PROVIDERS"); providersStr != "" {
        for _, p := range strings.Split(providersStr, ",") {
            if trimmed := strings.TrimSpace(p); trimmed != "" {
                providers = append(providers, trimmed)
            }
        }
    }
    return &Config{Providers: providers}, nil
}

// llm/llm.go
func getProviderPreferences() *ProviderPreferences {
    if len(config.Providers) == 0 {
        return nil  // Use OpenRouter default routing
    }
    log.Printf("LLM: Using provider preferences: order=%v, allow_fallbacks=false",
        config.Providers)
    return &ProviderPreferences{
        Order:          config.Providers,
        AllowFallbacks: false,
    }
}
```

**Logging Added:**
```
LLM: Initialized with 3 provider(s): [crusoe/bf16 novita/bf16 deepinfra/bf16]
LLM: Using provider preferences: order=[crusoe/bf16 novita/bf16 deepinfra/bf16], allow_fallbacks=false
LLM: API request includes provider preferences: &{Order:[...] AllowFallbacks:false}
LLM: API response status: 200 200 OK
```

## Consequences

### Positive

- **User control**: Explicit provider selection
- **Visibility**: Logging shows which providers are used
- **No fallbacks**: `allow_fallbacks=false` ensures chosen providers only
- **Performance optimization**: Users can prioritize fast providers
- **Cost optimization**: Users can prioritize cheap providers
- **Testable**: Easy to verify provider routing in logs

### Negative

- **Configuration complexity**: Users must know provider names
- **Case-sensitive**: Provider names must match exactly (e.g., `crusoe/bf16` not `Crusoe/BF16`)
- **No validation**: Typos in provider names cause API errors
- **Brittleness**: Provider availability changes over time

### Neutral

- Empty `PROVIDERS` uses OpenRouter default routing (recommended for most users)
- Comma-separated format is standard but requires careful parsing
- Alternative formats (JSON array) rejected for simplicity

## References

- Configuration: `PROVIDERS=crusoe/bf16,novita/bf16,deepinfra/bf16`
- OpenRouter docs: https://openrouter.ai/docs#provider-routing
- Log prefix: `LLM:` for all LLM-related messages
- Test files updated: All test files now pass `Providers` field
- Related: No retry logic (ADR-009 implied) - single attempt with chosen providers
