// Test program to validate popup countdown flow
package main

import (
	"fmt"
	"os"
	"time"

	"screen-ocr-llm/config"
	"screen-ocr-llm/llm"
	"screen-ocr-llm/ocr"
	"screen-ocr-llm/popup"
)

func main() {
	fmt.Println("=== Popup Flow Test ===")
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
	}
	fmt.Println()

	llm.Init(&llm.Config{
		APIKey:    cfg.APIKey,
		Model:     cfg.Model,
		Providers: cfg.Providers,
	})

	// Test 1: First OCR with countdown
	fmt.Println("=== Test 1: First OCR ===")
	testOCR("First")
	
	// Wait for popup to close
	fmt.Println("Waiting 5 seconds for popup to close...")
	time.Sleep(5 * time.Second)
	
	// Test 2: Second OCR with countdown (this is where the bug occurs)
	fmt.Println("\n=== Test 2: Second OCR (testing for WM_QUIT bug) ===")
	testOCR("Second")
	
	// Wait for popup to close
	fmt.Println("Waiting 5 seconds for popup to close...")
	time.Sleep(5 * time.Second)
	
	// Test 3: Third OCR to be sure
	fmt.Println("\n=== Test 3: Third OCR ===")
	testOCR("Third")
	
	fmt.Println("\n✓ All tests completed successfully!")
	fmt.Println("Check screen_ocr_debug.log for details")
}

func testOCR(testName string) {
	// Load test-image.png
	fmt.Printf("%s: Loading test-image.png...\n", testName)
	imageData, err := os.ReadFile("test-image.png")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: Failed to load test-image.png: %v\n", testName, err)
		os.Exit(1)
	}
	fmt.Printf("%s: Loaded %d bytes\n", testName, len(imageData))

	// Start countdown popup
	fmt.Printf("%s: Starting countdown popup (10 seconds)...\n", testName)
	err = popup.StartCountdown(10)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: Failed to start countdown: %v\n", testName, err)
		os.Exit(1)
	}
	
	// Perform OCR (simulating the worker)
	fmt.Printf("%s: Performing OCR...\n", testName)
	text, err := ocr.RecognizeImage(imageData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: OCR failed: %v\n", testName, err)
		popup.Close()
		os.Exit(1)
	}

	// Update popup with result
	fmt.Printf("%s: Updating popup with result (%d characters)...\n", testName, len(text))
	err = popup.UpdateText(text)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: Failed to update popup: %v\n", testName, err)
		os.Exit(1)
	}
	
	fmt.Printf("%s: ✓ Test completed\n", testName)
}

