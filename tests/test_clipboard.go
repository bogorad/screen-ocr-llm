package main

import (
	"fmt"
	"log"

	"golang.design/x/clipboard"
)

func main() {
	// Initialize clipboard
	err := clipboard.Init()
	if err != nil {
		log.Fatalf("Failed to initialize clipboard: %v", err)
	}
	fmt.Println("Clipboard initialized successfully")

	// Test writing to clipboard
	testText := "Hello from Go clipboard test!"
	fmt.Printf("Writing to clipboard: %q\n", testText)

	clipboard.Write(clipboard.FmtText, []byte(testText))
	fmt.Println("Text written to clipboard")

	// Test reading from clipboard
	fmt.Println("Reading from clipboard...")
	data := clipboard.Read(clipboard.FmtText)
	fmt.Printf("Read from clipboard: %q\n", string(data))

	if string(data) == testText {
		fmt.Println("✅ Clipboard test PASSED!")
	} else {
		fmt.Println("❌ Clipboard test FAILED!")
		fmt.Printf("Expected: %q\n", testText)
		fmt.Printf("Got: %q\n", string(data))
	}
}
