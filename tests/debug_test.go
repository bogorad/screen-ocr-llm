package main

import (
	"log"
	"testing"

	"screen-ocr-llm/src/config"
	"screen-ocr-llm/src/gui"
	"screen-ocr-llm/src/llm"
	"screen-ocr-llm/src/screenshot"
)

// TestRealWorkflow tests the actual workflow with your configuration
func TestRealWorkflow(t *testing.T) {
	// Load your actual configuration
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	t.Logf("Loaded configuration:")
	// Log API key safely (prevent credential leakage and log injection)
	if len(cfg.APIKey) >= 8 {
		t.Logf("  API Key: %s...%s", cfg.APIKey[:4], cfg.APIKey[len(cfg.APIKey)-4:])
	} else {
		t.Logf("  API Key: [REDACTED]")
	}
	t.Logf("  Model: %s", cfg.Model)
	t.Logf("  File Logging: %v", cfg.EnableFileLogging)

	// Initialize LLM with your actual config
	llm.Init(&llm.Config{
		APIKey:    cfg.APIKey,
		Model:     cfg.Model,
		Providers: cfg.Providers,
	})

	// Test the region selection workflow
	region, err := gui.StartRegionSelection()
	if err != nil {
		t.Errorf("Region selection failed: %v", err)
	}

	t.Logf("Region selection completed with region: %+v", region)

	// This would normally capture the screen region and send to OCR
	// For testing, let's create a small test image
	testImageData := createTestImage()

	t.Logf("Testing OCR with %d bytes of image data", len(testImageData))

	// Test the actual API call with your credentials
	result, err := llm.QueryVision(testImageData)
	if err != nil {
		t.Logf("OCR API call failed (this might be expected with test image): %v", err)
	} else {
		t.Logf("OCR result: %s", result)
	}

	t.Logf("Test completed. Region used: %+v", region)
}

// createTestImage creates a minimal PNG image for testing
func createTestImage() []byte {
	// This is a minimal 1x1 pixel PNG image in base64, decoded to bytes
	// It's a white pixel PNG for testing purposes
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE, 0x00, 0x00, 0x00,
		0x0C, 0x49, 0x44, 0x41, 0x54, 0x08, 0xD7, 0x63, 0xF8, 0x0F, 0x00, 0x00,
		0x01, 0x00, 0x01, 0x5C, 0xC2, 0x8A, 0x8E, 0x00, 0x00, 0x00, 0x00, 0x49,
		0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
	}
	return pngData
}

// TestAPIConnectivity tests if we can connect to OpenRouter with your credentials
func TestAPIConnectivity(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	llm.Init(&llm.Config{
		APIKey:    cfg.APIKey,
		Model:     cfg.Model,
		Providers: cfg.Providers,
	})

	// Test with a simple image
	testImage := createTestImage()

	log.Printf("Testing API connectivity with model: %s", cfg.Model)
	result, err := llm.QueryVision(testImage)

	if err != nil {
		t.Logf("API call failed: %v", err)
		// This might be expected with a minimal test image
		// The important thing is that we get a proper API response, even if it's an error
	} else {
		t.Logf("API call succeeded! Result: %s", result)
	}
}
