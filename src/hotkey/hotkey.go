package hotkey

import (
	"log"
	"strings"
	"sync"

	gohook "github.com/robotn/gohook"
)

func Listen(hotkeyConfig string, callback func()) {
	// Note: This function only registers the hotkey and calls the callback when pressed.
	// The callback is responsible for triggering the region selection and OCR workflow.
	// The OCR processing is now handled by the eventloop after region selection completes.

	// Parse hotkey configuration
	keys := parseHotkey(hotkeyConfig)
	log.Printf("Parsed hotkey configuration: %v", keys)

	// Build a map of rawcodes to key names for this hotkey combination
	type keyState struct {
		name     string
		rawcodes []uint16
		pressed  bool
	}

	var keyStates []keyState
	for _, keyName := range keys {
		rawcodes := keyNameToRawcodes(keyName)
		if len(rawcodes) == 0 {
			log.Printf("ERROR: Cannot map key '%s' to rawcodes, hotkey may not work correctly", keyName)
			continue
		}
		keyStates = append(keyStates, keyState{
			name:     keyName,
			rawcodes: rawcodes,
			pressed:  false,
		})
	}

	if len(keyStates) == 0 {
		log.Printf("ERROR: No valid keys in hotkey configuration '%s'", hotkeyConfig)
		return
	}

	log.Printf("Hotkey listener configured for: %s", hotkeyConfig)

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

					// Check if this rawcode matches any of our configured keys
					for i := range keyStates {
						for _, rawcode := range keyStates[i].rawcodes {
							if ev.Rawcode == rawcode {
								keyStates[i].pressed = true
								log.Printf("%s pressed", keyStates[i].name)
								break
							}
						}
					}

					// Check if all keys are pressed
					allPressed := true
					for i := range keyStates {
						if !keyStates[i].pressed {
							allPressed = false
							break
						}
					}

					if allPressed {
						log.Printf("HOTKEY COMBINATION DETECTED! %s", hotkeyConfig)
						log.Printf("Hotkey activated")
						// Reset states before releasing lock
						for i := range keyStates {
							keyStates[i].pressed = false
						}
						mu.Unlock()

						// Invoke the callback if provided
						// The callback is responsible for triggering the region selection workflow
						if callback != nil {
							callback()
						}
					} else {
						mu.Unlock()
					}
				} else if ev.Kind == gohook.KeyUp {
					mu.Lock()

					// Check if this rawcode matches any of our configured keys
					for i := range keyStates {
						for _, rawcode := range keyStates[i].rawcodes {
							if ev.Rawcode == rawcode {
								if keyStates[i].pressed {
									keyStates[i].pressed = false
									log.Printf("%s released", keyStates[i].name)
								}
								break
							}
						}
					}

					mu.Unlock()
				}
			}
		}
		log.Printf("Event channel closed")
	}()
}

// parseHotkey converts a hotkey string like "Ctrl+Alt+q" to normalized key names
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

// keyNameToRawcodes maps a key name to its Windows virtual key code rawcodes
// Returns a slice of rawcodes (e.g., both left and right variants for modifiers)
func keyNameToRawcodes(keyName string) []uint16 {
	keyName = strings.ToLower(strings.TrimSpace(keyName))

	switch keyName {
	// Modifier keys - return both left and right variants
	case "ctrl":
		return []uint16{162, 163} // VK_LCONTROL, VK_RCONTROL
	case "alt":
		return []uint16{164, 165} // VK_LMENU, VK_RMENU (MENU = Alt)
	case "shift":
		return []uint16{160, 161} // VK_LSHIFT, VK_RSHIFT
	case "win", "cmd", "super":
		return []uint16{91, 92} // VK_LWIN, VK_RWIN (Windows/Super/Cmd key)

	// Letter keys (A-Z) - VK codes 0x41-0x5A (65-90)
	case "a":
		return []uint16{65}
	case "b":
		return []uint16{66}
	case "c":
		return []uint16{67}
	case "d":
		return []uint16{68}
	case "e":
		return []uint16{69}
	case "f":
		return []uint16{70}
	case "g":
		return []uint16{71}
	case "h":
		return []uint16{72}
	case "i":
		return []uint16{73}
	case "j":
		return []uint16{74}
	case "k":
		return []uint16{75}
	case "l":
		return []uint16{76}
	case "m":
		return []uint16{77}
	case "n":
		return []uint16{78}
	case "o":
		return []uint16{79}
	case "p":
		return []uint16{80}
	case "q":
		return []uint16{81}
	case "r":
		return []uint16{82}
	case "s":
		return []uint16{83}
	case "t":
		return []uint16{84}
	case "u":
		return []uint16{85}
	case "v":
		return []uint16{86}
	case "w":
		return []uint16{87}
	case "x":
		return []uint16{88}
	case "y":
		return []uint16{89}
	case "z":
		return []uint16{90}

	// Number keys (0-9) - VK codes 0x30-0x39 (48-57)
	case "0":
		return []uint16{48}
	case "1":
		return []uint16{49}
	case "2":
		return []uint16{50}
	case "3":
		return []uint16{51}
	case "4":
		return []uint16{52}
	case "5":
		return []uint16{53}
	case "6":
		return []uint16{54}
	case "7":
		return []uint16{55}
	case "8":
		return []uint16{56}
	case "9":
		return []uint16{57}

	// Function keys (F1-F24)
	case "f1":
		return []uint16{112} // VK_F1
	case "f2":
		return []uint16{113} // VK_F2
	case "f3":
		return []uint16{114} // VK_F3
	case "f4":
		return []uint16{115} // VK_F4
	case "f5":
		return []uint16{116} // VK_F5
	case "f6":
		return []uint16{117} // VK_F6
	case "f7":
		return []uint16{118} // VK_F7
	case "f8":
		return []uint16{119} // VK_F8
	case "f9":
		return []uint16{120} // VK_F9
	case "f10":
		return []uint16{121} // VK_F10
	case "f11":
		return []uint16{122} // VK_F11
	case "f12":
		return []uint16{123} // VK_F12
	case "f13":
		return []uint16{124} // VK_F13
	case "f14":
		return []uint16{125} // VK_F14
	case "f15":
		return []uint16{126} // VK_F15
	case "f16":
		return []uint16{127} // VK_F16
	case "f17":
		return []uint16{128} // VK_F17
	case "f18":
		return []uint16{129} // VK_F18
	case "f19":
		return []uint16{130} // VK_F19
	case "f20":
		return []uint16{131} // VK_F20
	case "f21":
		return []uint16{132} // VK_F21
	case "f22":
		return []uint16{133} // VK_F22
	case "f23":
		return []uint16{134} // VK_F23
	case "f24":
		return []uint16{135} // VK_F24

	// Common special keys
	case "space":
		return []uint16{32} // VK_SPACE
	case "enter", "return":
		return []uint16{13} // VK_RETURN
	case "esc", "escape":
		return []uint16{27} // VK_ESCAPE
	case "tab":
		return []uint16{9} // VK_TAB
	case "backspace":
		return []uint16{8} // VK_BACK
	case "delete", "del":
		return []uint16{46} // VK_DELETE
	case "insert", "ins":
		return []uint16{45} // VK_INSERT
	case "home":
		return []uint16{36} // VK_HOME
	case "end":
		return []uint16{35} // VK_END
	case "pageup", "pgup":
		return []uint16{33} // VK_PRIOR
	case "pagedown", "pgdn":
		return []uint16{34} // VK_NEXT

	// Arrow keys
	case "left":
		return []uint16{37} // VK_LEFT
	case "up":
		return []uint16{38} // VK_UP
	case "right":
		return []uint16{39} // VK_RIGHT
	case "down":
		return []uint16{40} // VK_DOWN

	default:
		log.Printf("WARNING: Unknown key name '%s', cannot map to rawcode", keyName)
		return nil
	}
}
