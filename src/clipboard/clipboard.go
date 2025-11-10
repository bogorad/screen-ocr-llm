package clipboard

import (
	"golang.design/x/clipboard"
	"sync"
)

var (
	writeMu sync.Mutex
)

func Init() error {
	return clipboard.Init()
}

// Write performs a mutex-guarded clipboard write to prevent corruption under parallel writes.
func Write(text string) error {
	writeMu.Lock()
	defer writeMu.Unlock()
	clipboard.Write(clipboard.FmtText, []byte(text))
	return nil
}
