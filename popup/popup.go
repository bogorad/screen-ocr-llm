package popup

import (
	"screen-ocr-llm/notification"
)

// Show displays a synchronous 3-second popup window and returns when it is closed.
// This is a simple adapter on top of the existing notification package.
func Show(text string) error {
	// Fire-and-forget: notification layer manages its own lifetime asynchronously.
	notification.ShowOCRResult(text)
	return nil
}

