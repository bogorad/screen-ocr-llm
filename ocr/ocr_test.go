package ocr

import (
	"os"
	"testing"

	"screen-ocr-llm/llm"
	"screen-ocr-llm/screenshot"
)

func TestRecognize(t *testing.T) {
	// Get API key from environment variable
	apiKey := os.Getenv("TEST_API_KEY")
	if apiKey == "" {
		apiKey = "mock_key_for_error_testing" // Safe mock for error testing
	}

	// Initialize LLM with test config
	llm.Init(&llm.Config{
		APIKey: apiKey,
		Model:  "test_model",
	})

	// Test with invalid region (should fail at screenshot capture)
	region := screenshot.Region{X: 0, Y: 0, Width: 0, Height: 0}
	_, err := Recognize(region)
	if err == nil {
		t.Error("Expected error with invalid region")
	}
	t.Logf("OCR with invalid region failed as expected: %v", err)

	// Test with valid region (will fail due to no display or invalid API key)
	region = screenshot.Region{X: 0, Y: 0, Width: 100, Height: 100}
	_, err = Recognize(region)
	if err == nil {
		t.Error("Expected error (no display or invalid API key)")
	}
	t.Logf("OCR with valid region failed as expected: %v", err)
}

func TestRecognizeImage(t *testing.T) {
	// Get API key from environment variable
	apiKey := os.Getenv("TEST_API_KEY")
	if apiKey == "" {
		apiKey = "mock_key_for_error_testing" // Safe mock for error testing
	}

	// Initialize LLM with test config
	llm.Init(&llm.Config{
		APIKey: apiKey,
		Model:  "test_model",
	})

	// Test with image data (will fail due to invalid API key)
	testImageData := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	_, err := RecognizeImage(testImageData)
	if err == nil {
		t.Error("Expected error with invalid API key")
	}
	t.Logf("OCR with image data failed as expected: %v", err)
}
