package clipboard

import (
	"golang.design/x/clipboard"
)

func Init() error {
	return clipboard.Init()
}

func Write(text string) error {
	// Write to clipboard - this returns a channel, not an error
	clipboard.Write(clipboard.FmtText, []byte(text))
	return nil
}
