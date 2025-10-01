package hotkey

import (
	"os"
	"testing"

	"screen-ocr-llm/clipboard"
	"screen-ocr-llm/llm"
)

func TestListen(t *testing.T) {
	// Get API key from environment variable
	apiKey := os.Getenv("TEST_API_KEY")
	if apiKey == "" {
		t.Skip("TEST_API_KEY not set; skipping test")
	}

	// Initialize required packages for testing
	llm.Init(&llm.Config{
		APIKey:    apiKey,
		Model:     "test_model",
		Providers: []string{}, // Empty for test
	})

	err := clipboard.Init()
	if err != nil {
		t.Logf("Clipboard init failed (expected in headless environment): %v", err)
	}

	// This test would require user interaction, so we'll just check if the function exists
	// and doesn't panic during setup
	Listen("Ctrl+Alt+Q", func() {
		// Test callback - this won't be called in test environment
	})

	t.Log("Hotkey listener setup completed successfully")
}
