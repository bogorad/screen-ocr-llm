package messages

import (
	"screen-ocr-llm/src/config"
	"screen-ocr-llm/src/screenshot"
)

// Message is the base interface for all inter-process messages
type Message interface {
	Type() string
}

// MessageType constants for type identification
const (
	TypeHotkeyPressed     = "HotkeyPressed"
	TypeStartRegionSelect = "StartRegionSelection"
	TypeRegionSelected    = "RegionSelected"
	TypeRegionCancelled   = "RegionCancelled"
	TypeProcessRegion     = "ProcessRegion"
	TypeOCRComplete       = "OCRComplete"
	TypeWriteClipboard    = "WriteClipboard"
	TypeClipboardComplete = "ClipboardComplete"
	TypeShowPopup         = "ShowPopup"
	TypeUpdateTray        = "UpdateTray"
	TypeTrayMenuClicked   = "TrayMenuClicked"
	TypeConfigChanged     = "ConfigChanged"
	TypeRunOnceRequest    = "RunOnceRequest"
	TypeRunOnceComplete   = "RunOnceComplete"
	TypeDieNow            = "DIENOW"
)

// HotkeyPressed - sent by hotkey process when hotkey combination is detected
type HotkeyPressed struct {
	Combo string // e.g., "Ctrl+Alt+Q"
}

func (m HotkeyPressed) Type() string { return TypeHotkeyPressed }

// StartRegionSelection - sent to region selection process to begin region selection
type StartRegionSelection struct{}

func (m StartRegionSelection) Type() string { return TypeStartRegionSelect }

// RegionSelected - sent by region selection process when user selects a region
type RegionSelected struct {
	Region screenshot.Region
}

func (m RegionSelected) Type() string { return TypeRegionSelected }

// RegionCancelled - sent by region selection process when user cancels selection
type RegionCancelled struct{}

func (m RegionCancelled) Type() string { return TypeRegionCancelled }

// ProcessRegion - sent to OCR process to perform OCR on a specific region
type ProcessRegion struct {
	Region screenshot.Region
}

func (m ProcessRegion) Type() string { return TypeProcessRegion }

// OCRComplete - sent by OCR process when text extraction is complete
type OCRComplete struct {
	Text  string
	Error error
}

func (m OCRComplete) Type() string { return TypeOCRComplete }

// WriteClipboard - sent to clipboard process to write text to system clipboard
type WriteClipboard struct {
	Text string
}

func (m WriteClipboard) Type() string { return TypeWriteClipboard }

// ClipboardComplete - sent by clipboard process when clipboard operation is complete
type ClipboardComplete struct {
	Error error
}

func (m ClipboardComplete) Type() string { return TypeClipboardComplete }

// ShowPopup - sent to popup process to display a notification popup
type ShowPopup struct {
	Text     string // Text to display (will be truncated to 200 chars)
	Duration int    // Duration in seconds (0 = default 3 seconds)
}

func (m ShowPopup) Type() string { return TypeShowPopup }

// UpdateTray - sent to tray process to update tray icon status
type UpdateTray struct {
	Tooltip string
	Status  string // e.g., "idle", "processing", "error"
}

func (m UpdateTray) Type() string { return TypeUpdateTray }

// TrayMenuClicked - sent by tray process when user clicks a menu item
type TrayMenuClicked struct {
	Action string // e.g., "about", "exit"
}

func (m TrayMenuClicked) Type() string { return TypeTrayMenuClicked }

// ConfigChanged - sent by config process when configuration file changes
type ConfigChanged struct {
	Config config.Config
}

func (m ConfigChanged) Type() string { return TypeConfigChanged }

// RunOnceRequest - sent to resident main process to perform a single OCR operation
type RunOnceRequest struct {
	OutputToStdout bool // true for --run-once-std, false for --run-once
}

func (m RunOnceRequest) Type() string { return TypeRunOnceRequest }

// RunOnceComplete - sent back to requesting process when run-once operation is complete
type RunOnceComplete struct {
	Text  string // OCR result text
	Error error  // Error if operation failed
}

func (m RunOnceComplete) Type() string { return TypeRunOnceComplete }

// DIENOW - emergency shutdown message sent to all processes
type DIENOW struct{}

func (m DIENOW) Type() string { return TypeDieNow }

// MessageEnvelope wraps messages with metadata for routing
type MessageEnvelope struct {
	From    string  // Source process name
	To      string  // Destination process name ("*" for broadcast)
	Message Message // The actual message
}

// ProcessNames - constants for process identification
const (
	ProcessMain      = "main"
	ProcessHotkey    = "hotkey"
	ProcessRegionSel = "region"
	ProcessOCR       = "ocr"
	ProcessClipboard = "clipboard"
	ProcessPopup     = "popup"
	ProcessTray      = "tray"
	ProcessConfig    = "config"
)
