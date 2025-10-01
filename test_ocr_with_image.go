// Test program to validate OCR functionality with test-image.png
// This bypasses interactive region selection and directly tests the OCR pipeline
package main

import (
	"fmt"
	"os"

	"screen-ocr-llm/config"
	"screen-ocr-llm/llm"
	"screen-ocr-llm/ocr"
)

func main() {
	fmt.Println("=== OCR Test with test-image.png ===")
	fmt.Println()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize LLM
	fmt.Printf("Configuration loaded:\n")
	fmt.Printf("  Model: %s\n", cfg.Model)
	if len(cfg.Providers) > 0 {
		fmt.Printf("  Providers: %v\n", cfg.Providers)
	} else {
		fmt.Printf("  Providers: (none - using OpenRouter default routing)\n")
	}
	fmt.Println()

	llm.Init(&llm.Config{
		APIKey:    cfg.APIKey,
		Model:     cfg.Model,
		Providers: cfg.Providers,
	})

	// Load test-image.png
	fmt.Println("Loading test-image.png...")
	imageData, err := os.ReadFile("test-image.png")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load test-image.png: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Loaded %d bytes from test-image.png\n", len(imageData))
	fmt.Println()

	// Perform OCR
	fmt.Println("Performing OCR...")
	text, err := ocr.RecognizeImage(imageData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OCR failed: %v\n", err)
		os.Exit(1)
	}

	// Display result
	fmt.Println("=== OCR Result ===")
	fmt.Println(text)
	fmt.Println()
	fmt.Printf("Extracted %d characters\n", len(text))
	fmt.Println()
	fmt.Println("✓ Test completed successfully!")
}

