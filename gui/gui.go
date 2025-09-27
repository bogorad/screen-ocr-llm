package gui

import (
	"fmt"
	"log"

	"screen-ocr-llm/screenshot"

	"github.com/getlantern/systray"
)

// RegionSelectionCallback is called when a region is selected
type RegionSelectionCallback func(region screenshot.Region) error

var regionCallback RegionSelectionCallback

func Init() {
	// Initialize GUI package if needed
}

// SetRegionSelectionCallback sets the callback for region selection
func SetRegionSelectionCallback(callback RegionSelectionCallback) {
	regionCallback = callback
}

// StartRegionSelection starts the region selection process
func StartRegionSelection() error {
	if regionCallback == nil {
		return fmt.Errorf("no region selection callback set")
	}

	log.Printf("Starting interactive region selection...")

	// Use platform-specific region selection
	region, err := StartInteractiveRegionSelection()
	if err != nil {
		log.Printf("Interactive region selection failed: %v", err)
		return err
	}

	// Check if a valid region was selected
	if region.Width == 0 || region.Height == 0 {
		log.Printf("No valid region selected")
		return fmt.Errorf("no valid region selected")
	}

	log.Printf("Region selected: %+v", region)
	return regionCallback(region)
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
				if err := StartRegionSelection(); err != nil {
					log.Printf("Region selection failed: %v", err)
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
