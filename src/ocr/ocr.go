package ocr

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"screen-ocr-llm/src/llm"
	"screen-ocr-llm/src/screenshot"
)

func Init() {
	// Initialize OCR package if needed
}

// Recognize performs OCR on a screen region using OpenRouter vision models
func Recognize(region screenshot.Region) (string, error) {
	log.Printf("DEBUG: Capturing region: X=%d Y=%d Width=%d Height=%d", region.X, region.Y, region.Width, region.Height)

	// Capture the specified region
	imageData, err := screenshot.CaptureRegion(region)
	if err != nil {
		return "", err
	}

	// DEBUG: Save the captured image only if debug mode is enabled
	if os.Getenv("OCR_DEBUG_SAVE_IMAGES") == "true" {
		debugFilename := fmt.Sprintf("debug_captured_region_%dx%d.png", region.Width, region.Height)
		if err := ioutil.WriteFile(debugFilename, imageData, 0600); err != nil { // More restrictive permissions
			log.Printf("Warning: Could not save debug image: %v", err)
		} else {
			log.Printf("DEBUG: Saved captured region to %s (size: %d bytes)", debugFilename, len(imageData))
		}
	}

	// Send to OpenRouter vision model for OCR
	return llm.QueryVision(imageData)
}

// RecognizeImage performs OCR on provided image data using OpenRouter vision models
func RecognizeImage(imageData []byte) (string, error) {
	return llm.QueryVision(imageData)
}
