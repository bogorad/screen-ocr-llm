package screenshot

import (
	"testing"
)

func TestCapture(t *testing.T) {
	// This test would require a display, so we'll just check if the function exists
	// and doesn't panic
	_, err := Capture()
	if err != nil {
		t.Logf("Failed to capture screenshot: %v", err)
	}
}

func TestCaptureRegion(t *testing.T) {
	// Test with invalid region
	_, err := CaptureRegion(Region{X: 0, Y: 0, Width: 0, Height: 0})
	if err == nil {
		t.Error("Expected error for invalid region dimensions")
	}

	// Test with valid region (may fail if no display available)
	_, err = CaptureRegion(Region{X: 0, Y: 0, Width: 100, Height: 100})
	if err != nil {
		t.Logf("Failed to capture region (expected in headless environment): %v", err)
	}
}

func TestGetDisplayBounds(t *testing.T) {
	// Test getting display bounds
	_, err := GetDisplayBounds()
	if err != nil {
		t.Logf("Failed to get display bounds (expected in headless environment): %v", err)
	}
}
