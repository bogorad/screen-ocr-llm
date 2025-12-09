//go:build windows

package overlay

import (
	"context"
	"screen-ocr-llm/src/gui"
	"screen-ocr-llm/src/screenshot"
)

// windowsSelector adapts existing gui region selector to the new synchronous API.
type windowsSelector struct{}

func newWindowsSelector() Selector { return &windowsSelector{} }

func (w *windowsSelector) Select(ctx context.Context) (screenshot.Region, bool, error) {
	// Call gui.StartRegionSelection() directly - it now returns the region
	region, err := gui.StartRegionSelection()
	if err != nil {
		return screenshot.Region{}, false, err
	}

	// Check if context was cancelled during selection
	select {
	case <-ctx.Done():
		return screenshot.Region{}, false, ctx.Err()
	default:
		return region, false, nil
	}
}
