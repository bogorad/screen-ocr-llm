package llm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPingNotInitialized(t *testing.T) {
	activeConfig.Store(nil)
	if err := Ping(); err == nil {
		t.Error("Expected error when not initialized")
	}
}

func TestPingAcceptsReasoningOnlyResponseAndPreservesRouting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var request ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if request.Model != "nvidia/nemotron-3.5-super-vl:nitro" {
			t.Errorf("model changed: %q", request.Model)
		}
		if request.MaxTokens != 1 {
			t.Errorf("unexpected ping token budget: %d", request.MaxTokens)
		}
		if request.Provider == nil || len(request.Provider.Order) != 2 || request.Provider.Order[0] != "deepinfra/fp4" || request.Provider.Order[1] != "coreweave/nvfp4" {
			t.Errorf("provider order changed: %+v", request.Provider)
		}
		if request.Provider == nil || request.Provider.AllowFallbacks == nil || *request.Provider.AllowFallbacks {
			t.Errorf("fallback policy changed: %+v", request.Provider)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"","reasoning":"thinking"},"finish_reason":"length"}],"usage":{"completion_tokens":1,"completion_tokens_details":{"reasoning_tokens":1}}}`))
	}))
	defer server.Close()
	previousURL := openRouterURL
	openRouterURL = server.URL + "/api/v1/chat/completions"
	t.Cleanup(func() { openRouterURL = previousURL })
	Init(&Config{APIKey: "test-key", Model: "nvidia/nemotron-3.5-super-vl:nitro", Providers: []string{"deepinfra/fp4", "coreweave/nvfp4"}})
	if err := Ping(); err != nil {
		t.Fatalf("reasoning-only ping should prove API connectivity: %v", err)
	}
}

func TestQueryVision(t *testing.T) {
	// Test without initialization
	_, err := QueryVision([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	if err == nil {
		t.Error("Expected error when not initialized")
	}

	// Test with missing API key
	Init(&Config{
		APIKey:    "",
		Model:     "test_model",
		Providers: []string{},
	})
	_, err = QueryVision([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	if err == nil {
		t.Error("Expected error with missing API key")
	}

	// Test with missing model
	Init(&Config{
		APIKey:    "test_api_key",
		Model:     "",
		Providers: []string{},
	})
	_, err = QueryVision([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	if err == nil {
		t.Error("Expected error with missing model")
	}

	// Test with valid config (will fail due to invalid API key, but tests the request structure)
	Init(&Config{
		APIKey:    "mock_key_for_error_testing", // Safe mock for error testing
		Model:     "test_model",
		Providers: []string{},
	})
	testImageData := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	_, err = QueryVision(testImageData)
	if err == nil {
		t.Error("Expected error with invalid API key")
	}
	t.Logf("QueryVision validation working as expected: %v", err)
}
