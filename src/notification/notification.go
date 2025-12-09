package notification

import (
	"log"
	"runtime"
)

// ShowOCRResult displays a temporary popup with OCR results
func ShowOCRResult(text string) {
	// Truncate text to 200 characters
	displayText := text
	if len(text) > 200 {
		displayText = text[:200] + "..."
	}

	// Show platform-specific notification
	if runtime.GOOS == "windows" {
		showWindowsNotification(displayText)
	} else {
		// For other platforms, just log for now
		log.Printf("OCR Result: %s", displayText)
	}
}

// showWindowsNotification shows a notification on Windows
func showWindowsNotification(text string) {
	go func() {
		err := showWindowsPopup(text)
		if err != nil {
			log.Printf("Failed to show notification: %v", err)
		}
	}()
}

// showWindowsPopup is implemented in notification_windows.go
