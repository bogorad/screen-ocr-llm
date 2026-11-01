package logutil

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

const (
	logFileName  = "screen_ocr_debug.log"
	maxSizeBytes = 10 * 1024 * 1024 // 10 MB
	maxArchives  = 3
)

var setupMu sync.Mutex
var activeWriter *rotatingWriter

// Setup enables file logging with basic size-based rotation (10MB, max 3 files).
// When disabled, logs are discarded (keeps stdout clean) to match prior behavior.
func Setup(enableFileLogging bool) error {
	setupMu.Lock()
	defer setupMu.Unlock()
	if !enableFileLogging {
		log.SetOutput(io.Discard)
		if activeWriter != nil {
			_ = activeWriter.f.Close()
			activeWriter = nil
		}
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		return nil
	}
	rotateIfNeeded()
	f, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		return err
	}
	previous := activeWriter
	activeWriter = &rotatingWriter{f: f}
	log.SetOutput(activeWriter)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if previous != nil {
		_ = previous.f.Close()
	}
	return nil
}

type rotatingWriter struct{ f *os.File }

func (w *rotatingWriter) Write(p []byte) (int, error) {
	// naive rotation check per write
	if st, err := w.f.Stat(); err == nil && st.Size()+int64(len(p)) > maxSizeBytes {
		_ = w.f.Close()
		rotateIfNeeded()
		nf, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return 0, err
		}
		w.f = nf
	}
	return w.f.Write(p)
}

func rotateIfNeeded() {
	// If base exceeds max size, rotate: .1, .2, .3 (oldest discarded)
	if st, err := os.Stat(logFileName); err == nil && st.Size() > maxSizeBytes {
		// remove oldest
		_ = os.Remove(archiveName(maxArchives))
		// shift others
		for i := maxArchives - 1; i >= 1; i-- {
			_ = os.Rename(archiveName(i), archiveName(i+1))
		}
		// move current to .1
		_ = os.Rename(logFileName, archiveName(1))
	}
}

func archiveName(n int) string { return filepath.Join(".", fmt.Sprintf("%s.%d", logFileName, n)) }

// RedactKey masks an API key, leaving first/last 4 chars: xxxx...yyyy
func RedactKey(k string) string {
	if len(k) <= 8 {
		return "********"
	}
	return fmt.Sprintf("%s...%s", k[:4], k[len(k)-4:])
}
