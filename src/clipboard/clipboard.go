package clipboard

import (
	"errors"
	"sync"

	"golang.design/x/clipboard"
)

var (
	writeMu   sync.Mutex
	writeText = func(text string) <-chan struct{} {
		return clipboard.Write(clipboard.FmtText, []byte(text))
	}
)

func Init() error {
	return clipboard.Init()
}

// Write performs a mutex-guarded clipboard write to prevent corruption under parallel writes.
func Write(text string) error {
	writeMu.Lock()
	defer writeMu.Unlock()
	if writeText(sanitizeText(text)) == nil {
		return errors.New("clipboard write failed")
	}
	return nil
}

func sanitizeText(text string) string {
	clean := make([]rune, 0, len(text))
	for _, r := range text {
		if r == '\n' || r == '\r' || r == '\t' {
			clean = append(clean, r)
			continue
		}
		if r < 32 || r == 127 || (r >= 0x80 && r <= 0x9f) {
			continue
		}
		clean = append(clean, r)
	}
	return string(clean)
}
