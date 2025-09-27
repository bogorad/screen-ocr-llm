//go:build windows

package overlay

import (
	"context"
	"screen-ocr-llm/gui"
	"screen-ocr-llm/screenshot"
)

// windowsSelector adapts existing gui region selector to the new synchronous API.
type windowsSelector struct{}

func newWindowsSelector() Selector { return &windowsSelector{} }

func (w *windowsSelector) Select(ctx context.Context) (screenshot.Region, bool, error) {
	var (
		selected screenshot.Region
		got bool
		done = make(chan struct{}, 1)
		err error
	)

	// Bridge: use existing gui package callbacks to obtain a region synchronously.
	gui.SetRegionSelectionCallback(func(r screenshot.Region) error {
		selected = r
		got = true
		done <- struct{}{}
		return nil
	})
	if e := gui.StartRegionSelection(); e != nil {
		return screenshot.Region{}, false, e
	}

	select {
	case <-done:
		if got {
			return selected, false, nil
		}
		return screenshot.Region{}, true, nil // cancelled
	case <-ctx.Done():
		err = ctx.Err()
	}

	return screenshot.Region{}, false, err
}

