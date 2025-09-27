package llm

import (
	"os"
	"testing"
)



func TestQueryVision(t *testing.T) {
	// Test without initialization
	_, err := QueryVision([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	if err == nil {
		t.Error("Expected error when not initialized")
	}

	// Test with missing API key
	Init(&Config{
		APIKey: "",
		Model:  "test_model",
	})
	_, err = QueryVision([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	if err == nil {
		t.Error("Expected error with missing API key")
	}

	// Test with missing model
	Init(&Config{
		APIKey: "test_api_key",
		Model:  "",
	})
	_, err = QueryVision([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	if err == nil {
		t.Error("Expected error with missing model")
	}

	// Test with valid config (will fail due to invalid API key, but tests the request structure)
	Init(&Config{
		APIKey: "mock_key_for_error_testing", // Safe mock for error testing
		Model:  "test_model",
	})
	testImageData := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	_, err = QueryVision(testImageData)
	if err == nil {
		t.Error("Expected error with invalid API key")
	}
	t.Logf("QueryVision validation working as expected: %v", err)
}
