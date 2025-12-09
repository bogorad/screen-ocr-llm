package popup

import (
	"log"
	"runtime"
	"screen-ocr-llm/src/notification"
)

// Show displays a synchronous 3-second popup window and returns when it is closed.
// This is a simple adapter on top of the existing notification package.
func Show(text string) error {
	// Get caller information for debugging
	_, file, line, ok := runtime.Caller(1)
	if ok {
		log.Printf("Popup.Show called from %s:%d with %d characters: %q", file, line, len(text), truncateForLog(text, 50))
	} else {
		log.Printf("Popup.Show called with %d characters: %q", len(text), truncateForLog(text, 50))
	}
	// Fire-and-forget: notification layer manages its own lifetime asynchronously.
	notification.ShowOCRResult(text)
	return nil
}

func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// StartCountdown displays a countdown popup that updates every second
func StartCountdown(timeoutSeconds int) error {
	log.Printf("Popup.StartCountdown called with %d seconds", timeoutSeconds)
	return notification.StartCountdownPopup(timeoutSeconds)
}

// UpdateText updates the text of the current popup (switches from countdown to result)
func UpdateText(text string) error {
	log.Printf("Popup.UpdateText called with %d characters", len(text))
	return notification.UpdatePopupText(text)
}

// Close closes the current popup
func Close() error {
	log.Printf("Popup.Close called")
	return notification.ClosePopup()
}
