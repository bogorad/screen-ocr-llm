package gui

import (
	"testing"

	"screen-ocr-llm/screenshot"
)

func TestInit(t *testing.T) {
	// Test that Init doesn't panic
	Init()
}

func TestSetRegionSelectionCallback(t *testing.T) {
	// Test setting callback
	called := false
	callback := func(region screenshot.Region) error {
		called = true
		return nil
	}

	SetRegionSelectionCallback(callback)

	// Test region selection
	err := StartRegionSelection()
	if err != nil {
		t.Errorf("StartRegionSelection failed: %v", err)
	}

	if !called {
		t.Error("Callback was not called")
	}
}

func TestStartRegionSelectionWithoutCallback(t *testing.T) {
	// Reset callback
	regionCallback = nil

	// Test without callback
	err := StartRegionSelection()
	if err == nil {
		t.Error("Expected error when no callback is set")
	}
}
