package screenshot

import (
	"image"
	"image/color"
	"testing"
)

func TestPointInPolygon(t *testing.T) {
	poly := []Point{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10}}

	if !pointInPolygon(5.5, 5.5, poly) {
		t.Fatal("expected center point to be inside polygon")
	}
	if pointInPolygon(-1, 5, poly) {
		t.Fatal("expected point outside polygon to be outside")
	}
	if !pointInPolygon(0, 5, poly) {
		t.Fatal("expected edge point to be treated as inside")
	}
}

func TestApplyPolygonMaskWhitesOutsidePixels(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 6, 6))
	black := color.RGBA{R: 0, G: 0, B: 0, A: 255}
	for y := 0; y < 6; y++ {
		for x := 0; x < 6; x++ {
			img.SetRGBA(x, y, black)
		}
	}

	region := Region{
		X:      50,
		Y:      80,
		Width:  6,
		Height: 6,
		Polygon: []Point{
			{X: 51, Y: 81},
			{X: 54, Y: 81},
			{X: 54, Y: 84},
			{X: 51, Y: 84},
		},
	}

	applyPolygonMask(img, region)

	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	if got := img.RGBAAt(0, 0); got != white {
		t.Fatalf("expected outside pixel to be white, got %#v", got)
	}
	if got := img.RGBAAt(2, 2); got != black {
		t.Fatalf("expected inside pixel to remain original color, got %#v", got)
	}
	if got := img.RGBAAt(1, 2); got != black {
		t.Fatalf("expected edge pixel to remain original color, got %#v", got)
	}
	if got := img.RGBAAt(5, 5); got != white {
		t.Fatalf("expected outside corner pixel to be white, got %#v", got)
	}
}
