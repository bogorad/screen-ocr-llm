//go:build !windows

package notification

import "log"

// ShowBlockingError logs a blocking error message on non-Windows platforms.
func ShowBlockingError(title, message string) {
	log.Printf("%s: %s", title, message)
}

func showWindowsPopup(text string) error {
	log.Printf("OCR Result: %s", text)
	return nil
}
