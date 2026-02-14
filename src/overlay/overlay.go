package overlay

import (
	"context"
	"screen-ocr-llm/src/screenshot"
)

// Selector defines a synchronous region-selection API owned by the event loop.
// The call is blocking and MUST be invoked only from the single event-loop goroutine.
// Returns (region, cancelled, error). If cancelled is true, region is undefined and err is nil.
type Selector interface {
	Select(ctx context.Context) (screenshot.Region, bool, error)
}

// NewSelector returns the platform implementation (Windows in this project).
// Implementation is provided in a platform-specific file.
func NewSelector(defaultMode string) Selector {
	return newWindowsSelector(defaultMode)
}
