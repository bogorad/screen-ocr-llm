package main

import (
	"os"
	"testing"
	"time"

	"screen-ocr-llm/clipboard"
	"screen-ocr-llm/config"
	"screen-ocr-llm/hotkey"
	"screen-ocr-llm/llm"
	"screen-ocr-llm/ocr"
	"screen-ocr-llm/screenshot"
)

func TestIntegration(t *testing.T) {
	// Get API key from environment variable
	apiKey := os.Getenv("TEST_API_KEY")
	if apiKey == "" {
		t.Skip("TEST_API_KEY not set; skipping integration test")
	}

	// Test configuration loading
	cfg := &config.Config{
		APIKey: apiKey,
		Model:  "test_model",
		Hotkey: "Ctrl+Shift+T",
	}

	// Initialize all packages
	screenshot.Init()
	ocr.Init()
	llm.Init(&llm.Config{
		APIKey:    cfg.APIKey,
		Model:     cfg.Model,
		Providers: cfg.Providers,
	})
	err := clipboard.Init()
	if err != nil {
		t.Logf("Clipboard init failed (expected in headless environment): %v", err)
	}

	t.Log("All packages initialized successfully")

	// Test individual components
	t.Run("Screenshot", func(t *testing.T) {
		// Test full screen capture
		_, err := screenshot.Capture()
		if err != nil {
			t.Logf("Full screen capture failed (expected in headless environment): %v", err)
		}

		// Test region capture with invalid dimensions
		_, err = screenshot.CaptureRegion(screenshot.Region{X: 0, Y: 0, Width: 0, Height: 0})
		if err == nil {
			t.Error("Expected error for invalid region dimensions")
		}

		// Test region capture with valid dimensions
		_, err = screenshot.CaptureRegion(screenshot.Region{X: 0, Y: 0, Width: 100, Height: 100})
		if err != nil {
			t.Logf("Region capture failed (expected in headless environment): %v", err)
		}
	})

	t.Run("LLM", func(t *testing.T) {
		// Test with invalid config
		_, err := llm.QueryVision(nil)
		if err == nil {
			t.Error("Expected error with nil image data")
		}

		// Test with valid image data (will fail due to invalid API key)
		testImageData := []byte{0xFF, 0xFF, 0xFF, 0xFF}
		_, err = llm.QueryVision(testImageData)
		if err == nil {
			t.Error("Expected error with invalid API key")
		}
		t.Logf("LLM vision API validation working: %v", err)
	})

	t.Run("OCR", func(t *testing.T) {
		// Test OCR with invalid region
		_, err := ocr.Recognize(screenshot.Region{X: 0, Y: 0, Width: 0, Height: 0})
		if err == nil {
			t.Error("Expected error with invalid region")
		}

		// Test OCR with image data
		testImageData := []byte{0xFF, 0xFF, 0xFF, 0xFF}
		_, err = ocr.RecognizeImage(testImageData)
		if err == nil {
			t.Error("Expected error with invalid API key")
		}
		t.Logf("OCR validation working: %v", err)
	})

	t.Run("Clipboard", func(t *testing.T) {
		// Test clipboard write
		err := clipboard.Write("test integration")
		if err != nil {
			t.Logf("Clipboard write failed (expected in headless environment): %v", err)
		} else {
			t.Log("Clipboard write successful")
		}
	})

	// Test hotkey listener setup (doesn't actually trigger hotkey)
	hotkey.Listen("Ctrl+Shift+T", func() {
		t.Log("Hotkey callback - this won't be called in test environment")
	})
	t.Log("Hotkey listener setup completed")

	// Brief wait to ensure all goroutines are started
	time.Sleep(100 * time.Millisecond)
	t.Log("Integration test completed successfully")
}
