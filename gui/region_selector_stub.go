//go:build !windows

package gui

import (
	"fmt"
	"screen-ocr-llm/screenshot"
)

// StartInteractiveRegionSelection is a stub for non-Windows platforms
func StartInteractiveRegionSelection() (screenshot.Region, error) {
	return screenshot.Region{}, fmt.Errorf("interactive region selection not implemented for this platform")
}
