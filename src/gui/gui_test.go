package gui

import (
	"testing"
)

func TestInit(t *testing.T) {
	// Test that Init doesn't panic
	Init()
}

func TestStartRegionSelection(t *testing.T) {
	// Test region selection
	// Note: This will open an interactive overlay window
	// In a real test environment, you would mock StartInteractiveRegionSelection
	region, err := StartRegionSelection()
	if err != nil {
		t.Errorf("StartRegionSelection failed: %v", err)
	}

	// Check that a valid region was returned
	if region.Width == 0 || region.Height == 0 {
		t.Error("Expected valid region with non-zero dimensions")
	}
}
