package screenshot

import (
	"bytes"
	"fmt"
	"image"
	"image/png"

	"github.com/kbinani/screenshot"
)

// Region represents a screen region to capture
type Region struct {
	X      int
	Y      int
	Width  int
	Height int
}

func Init() {
	// Initialize screenshot package if needed
}

// Capture captures the entire virtual screen across all active displays
func Capture() (*image.RGBA, error) {
	n := screenshot.NumActiveDisplays()
	if n == 0 {
		return nil, fmt.Errorf("no active displays found")
	}
	// Compute union of all display bounds
	union := screenshot.GetDisplayBounds(0)
	for i := 1; i < n; i++ {
		b := screenshot.GetDisplayBounds(i)
		union = union.Union(b)
	}
	// Capture the union rectangle
	img, err := screenshot.CaptureRect(union)
	if err != nil {
		return nil, err
	}
	return img, nil
}

// CaptureRegion captures a specific region of the screen
func CaptureRegion(region Region) ([]byte, error) {
	// Validate region
	if region.Width <= 0 || region.Height <= 0 {
		return nil, fmt.Errorf("invalid region dimensions: width=%d, height=%d", region.Width, region.Height)
	}

	// Create bounds for the region
	bounds := image.Rect(region.X, region.Y, region.X+region.Width, region.Y+region.Height)

	// Capture the region
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, fmt.Errorf("failed to capture region: %v", err)
	}

	// Convert to PNG bytes
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode image as PNG: %v", err)
	}

	return buf.Bytes(), nil
}

// GetDisplayBounds returns the bounds of the primary display
func GetDisplayBounds() (image.Rectangle, error) {
	n := screenshot.NumActiveDisplays()
	if n == 0 {
		return image.Rectangle{}, fmt.Errorf("no active displays found")
	}

	// Get bounds of the primary display (display 0)
	bounds := screenshot.GetDisplayBounds(0)
	return bounds, nil
}
