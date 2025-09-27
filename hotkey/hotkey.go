package hotkey

import (
	"log"
	"strings"
	"sync"
	"time"

	"screen-ocr-llm/clipboard"
	"screen-ocr-llm/gui"
	"screen-ocr-llm/notification"
	"screen-ocr-llm/ocr"
	"screen-ocr-llm/screenshot"

	gohook "github.com/robotn/gohook"
)

func Listen(hotkeyConfig string, callback func()) {
	// Set up the region selection callback to handle the complete OCR workflow
	gui.SetRegionSelectionCallback(func(region screenshot.Region) error {
		log.Printf("Processing region: %+v", region)

		// Perform OCR on the selected region
		text, err := ocr.Recognize(region)
		if err != nil {
			log.Printf("OCR failed: %v", err)
			return err
		}

		// Log OCR result safely (prevent log injection)
		safeText := sanitizeForLogging(text)
		log.Printf("OCR extracted text (%d chars): %q", len(text), safeText)

		// Copy result to clipboard
		if err := clipboard.Write(text); err != nil {
			log.Printf("CLIPBOARD ERROR: Failed to write to clipboard: %v", err)
			return err
		}

		log.Printf("OCR completed successfully, text copied to clipboard (%d chars)", len(text))

		// Show notification popup with OCR result
		notification.ShowOCRResult(text)

		return nil
	})

	// Parse hotkey configuration
	keys := parseHotkey(hotkeyConfig)
	log.Printf("Parsed hotkey configuration: %v", keys)

	// Start a goroutine to listen for hotkey events
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in hotkey goroutine: %v", r)
			}
		}()

		log.Printf("Starting gohook goroutine...")

		// Track key states for combination detection with mutex protection
		var mu sync.Mutex
		var ctrlPressed, altPressed, qPressed bool

		// Start the event loop
		log.Printf("Starting gohook event loop...")
		evChan := gohook.Start()
		if evChan == nil {
			log.Printf("ERROR: gohook.Start() returned nil channel")
			return
		}
		log.Printf("gohook.Start() returned channel successfully")

		// Process events from the channel
		for ev := range evChan {
			// Only log key events, not mouse events to reduce spam
			if ev.Kind == gohook.KeyDown || ev.Kind == gohook.KeyUp {
				log.Printf("Key event: Kind=%v, Rawcode=%d, Keychar=%v", ev.Kind, ev.Rawcode, ev.Keychar)

				// Track key states with mutex protection
				if ev.Kind == gohook.KeyDown {
					mu.Lock()
					switch ev.Rawcode {
					case 162, 163: // Left/Right Ctrl
						ctrlPressed = true
						log.Printf("Ctrl pressed")
					case 164, 165: // Left/Right Alt
						altPressed = true
						log.Printf("Alt pressed")
					case 81: // Q key
						qPressed = true
						log.Printf("Q pressed")
					}

					// Check if all keys are pressed
					if ctrlPressed && altPressed && qPressed {
						log.Printf("HOTKEY COMBINATION DETECTED! Ctrl+Alt+Q")
						log.Printf("Hotkey activated - starting region selection")
						// Reset states before releasing lock
						ctrlPressed, altPressed, qPressed = false, false, false
						mu.Unlock()

						// Start region selection in a separate goroutine to avoid blocking the hotkey loop
						go func() {
							// Small delay to ensure keys are fully released before starting region selection
							time.Sleep(100 * time.Millisecond)
							log.Printf("Starting region selection after key release delay")
							if err := gui.StartRegionSelection(); err != nil {
								log.Printf("Region selection failed: %v", err)
							}
						}()
					} else {
						mu.Unlock()
					}
				} else if ev.Kind == gohook.KeyUp {
					mu.Lock()
					switch ev.Rawcode {
					case 162, 163: // Left/Right Ctrl
						if ctrlPressed {
							ctrlPressed = false
							log.Printf("Ctrl released")
						}
					case 164, 165: // Left/Right Alt
						if altPressed {
							altPressed = false
							log.Printf("Alt released")
						}
					case 81: // Q key
						if qPressed {
							qPressed = false
							log.Printf("Q released")
						}
					}
					mu.Unlock()
				}
			}
		}
		log.Printf("Event channel closed")
	}()
}

// parseHotkey converts a hotkey string like "Ctrl+Alt+q" to gohook format
func parseHotkey(hotkeyConfig string) []string {
	// Convert to lowercase and split by +
	parts := strings.Split(strings.ToLower(hotkeyConfig), "+")
	var keys []string

	for _, part := range parts {
		part = strings.TrimSpace(part)
		switch part {
		case "ctrl":
			keys = append(keys, "ctrl")
		case "alt":
			keys = append(keys, "alt")
		case "shift":
			keys = append(keys, "shift")
		case "win", "cmd", "super":
			keys = append(keys, "cmd")
		default:
			// Regular key
			keys = append(keys, part)
		}
	}

	return keys
}

// sanitizeForLogging removes potentially dangerous characters from text for safe logging
func sanitizeForLogging(text string) string {
	// Limit length to prevent log flooding
	const maxLogLength = 100
	if len(text) > maxLogLength {
		text = text[:maxLogLength] + "..."
	}

	// Replace newlines and other control characters to prevent log injection
	sanitized := ""
	for _, r := range text {
		if r == '\n' || r == '\r' {
			sanitized += "\\n"
		} else if r == '\t' {
			sanitized += "\\t"
		} else if r < 32 || r == 127 { // Control characters
			sanitized += "?"
		} else {
			sanitized += string(r)
		}
	}
	return sanitized
}
