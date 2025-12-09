package gui

import (
	"fmt"
	"log"

	"screen-ocr-llm/src/screenshot"

	"github.com/getlantern/systray"
)

func Init() {
	// Initialize GUI package if needed
}

// StartRegionSelection starts the region selection process and returns the selected region
func StartRegionSelection() (screenshot.Region, error) {
	log.Printf("Starting interactive region selection...")

	// Use platform-specific region selection
	region, err := StartInteractiveRegionSelection()
	if err != nil {
		log.Printf("Interactive region selection failed: %v", err)
		return screenshot.Region{}, err
	}

	// Check if a valid region was selected
	if region.Width == 0 || region.Height == 0 {
		log.Printf("No valid region selected")
		return screenshot.Region{}, fmt.Errorf("no valid region selected")
	}

	log.Printf("Region selected: %+v", region)
	return region, nil
}

func StartSystray() {
	// Start the systray
	systray.Run(onReady, onExit)
}

func onReady() {
	// Set the icon for the systray
	systray.SetIcon(getIcon())

	// Set the title and tooltip for the systray
	systray.SetTitle("Screen OCR LLM")
	systray.SetTooltip("Screen OCR LLM")

	// Add menu items
	mCapture := systray.AddMenuItem("Capture Screen", "Capture the screen")
	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	// Handle menu item events
	go func() {
		for {
			select {
			case <-mCapture.ClickedCh:
				region, err := StartRegionSelection()
				if err != nil {
					log.Printf("Region selection failed: %v", err)
				} else {
					log.Printf("Region captured: %+v", region)
					// Note: In the eventloop architecture, OCR processing is handled
					// by the eventloop after Select() returns. This systray menu
					// is a legacy interface and may need additional integration.
				}
			case <-mQuit.ClickedCh:
				systray.Quit()
			}
		}
	}()
}

func onExit() {
	// Clean up resources when the systray exits
}

func getIcon() []byte {
	// TODO: Return the icon data
	return nil
}
