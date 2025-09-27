package hotkey

import (
	"log"
	"strings"

	"screen-ocr-llm/clipboard"
	"screen-ocr-llm/gui"
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

		log.Printf("OCR extracted text (%d chars): %q", len(text), text)

		// Copy result to clipboard
		if err := clipboard.Write(text); err != nil {
			log.Printf("CLIPBOARD ERROR: Failed to write to clipboard: %v", err)
			return err
		}

		log.Printf("OCR completed successfully, text copied to clipboard (%d chars)", len(text))
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

		// Track key states for combination detection
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

				// Track key states
				if ev.Kind == gohook.KeyDown {
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
						if err := gui.StartRegionSelection(); err != nil {
							log.Printf("Region selection failed: %v", err)
						}
						// Reset states
						ctrlPressed, altPressed, qPressed = false, false, false
					}
				} else if ev.Kind == gohook.KeyUp {
					switch ev.Rawcode {
					case 162, 163: // Left/Right Ctrl
						ctrlPressed = false
						log.Printf("Ctrl released")
					case 164, 165: // Left/Right Alt
						altPressed = false
						log.Printf("Alt released")
					case 81: // Q key
						qPressed = false
						log.Printf("Q released")
					}
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
