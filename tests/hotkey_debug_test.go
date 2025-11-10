package main

import (
	"log"
	"testing"
	"time"

	"screen-ocr-llm/src/hotkey"
)

// TestHotkeyDebug tests if the hotkey system is working
func TestHotkeyDebug(t *testing.T) {
	log.Println("Testing hotkey system...")

	triggered := false

	// Set up hotkey listener
	hotkey.Listen("Ctrl+Shift+O", func() {
		log.Println("HOTKEY TRIGGERED!")
		triggered = true
	})

	log.Println("Hotkey listener started. Press Ctrl+Shift+O within 10 seconds...")

	// Wait for 10 seconds to see if hotkey is triggered
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		if triggered {
			t.Log("SUCCESS: Hotkey was detected!")
			return
		}
		log.Printf("Waiting... %d seconds remaining", 10-i)
	}

	t.Log("No hotkey detected within 10 seconds")
	t.Log("This could mean:")
	t.Log("1. Another application is intercepting the hotkey")
	t.Log("2. The hotkey library isn't working on this system")
	t.Log("3. The hotkey combination is different than expected")
}
