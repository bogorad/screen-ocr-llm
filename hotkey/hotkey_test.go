package hotkey

import (
	"testing"

	"screen-ocr-llm/clipboard"
	"screen-ocr-llm/llm"
)

func TestListen(t *testing.T) {
	// Initialize required packages for testing
	llm.Init(&llm.Config{
		APIKey: "test_api_key",
		Model:  "test_model",
	})

	err := clipboard.Init()
	if err != nil {
		t.Logf("Clipboard init failed (expected in headless environment): %v", err)
	}

	// This test would require user interaction, so we'll just check if the function exists
	// and doesn't panic during setup
	Listen(func() {
		// Test callback - this won't be called in test environment
	})

	t.Log("Hotkey listener setup completed successfully")
}
