package gui

import (
	"os"
	"runtime"
	"testing"
)

func TestInit(t *testing.T) {
	// Test that Init doesn't panic
	Init()
}

func TestStartRegionSelection(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("interactive region selection test is Windows-only")
	}
	if os.Getenv("SCREEN_OCR_INTERACTIVE_TESTS") != "1" {
		t.Skip("set SCREEN_OCR_INTERACTIVE_TESTS=1 to run interactive region selection test")
	}

	region, err := StartRegionSelection()
	if err != nil {
		t.Errorf("StartRegionSelection failed: %v", err)
	}

	// Check that a valid region was returned
	if region.Width == 0 || region.Height == 0 {
		t.Error("Expected valid region with non-zero dimensions")
	}
}
