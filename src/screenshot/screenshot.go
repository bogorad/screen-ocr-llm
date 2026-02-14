package screenshot

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"

	"github.com/kbinani/screenshot"
)

// Region represents a screen region to capture
type Region struct {
	X      int
	Y      int
	Width  int
	Height int
	// Polygon is optional and, when present, is expressed in absolute
	// virtual-screen coordinates. CaptureRegion uses it to mask pixels
	// outside the polygon while still returning a rectangular image.
	Polygon []Point
}

type Point struct {
	X int
	Y int
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

	if len(region.Polygon) >= 3 {
		applyPolygonMask(img, region)
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

func applyPolygonMask(img *image.RGBA, region Region) {
	localPolygon := make([]Point, len(region.Polygon))
	for i, p := range region.Polygon {
		localPolygon[i] = Point{X: p.X - region.X, Y: p.Y - region.Y}
	}

	b := img.Bounds()
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if !pointInPolygon(float64(x)+0.5, float64(y)+0.5, localPolygon) {
				img.SetRGBA(x, y, white)
			}
		}
	}
}

func pointInPolygon(px, py float64, polygon []Point) bool {
	if len(polygon) < 3 {
		return false
	}

	inside := false
	for i, j := 0, len(polygon)-1; i < len(polygon); j, i = i, i+1 {
		xi := float64(polygon[i].X)
		yi := float64(polygon[i].Y)
		xj := float64(polygon[j].X)
		yj := float64(polygon[j].Y)

		if pointOnSegment(px, py, xi, yi, xj, yj) {
			return true
		}

		intersects := ((yi > py) != (yj > py)) &&
			(px < (xj-xi)*(py-yi)/(yj-yi)+xi)
		if intersects {
			inside = !inside
		}
	}

	return inside
}

func pointOnSegment(px, py, x1, y1, x2, y2 float64) bool {
	const epsilon = 0.5
	cross := (px-x1)*(y2-y1) - (py-y1)*(x2-x1)
	if math.Abs(cross) > epsilon {
		return false
	}

	minX := math.Min(x1, x2) - epsilon
	maxX := math.Max(x1, x2) + epsilon
	minY := math.Min(y1, y2) - epsilon
	maxY := math.Max(y1, y2) + epsilon
	return px >= minX && px <= maxX && py >= minY && py <= maxY
}
