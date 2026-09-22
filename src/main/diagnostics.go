package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// reportApplicationError makes fatal errors visible in GUI builds, where
// stderr is hidden and routine file logging may be disabled.
func reportApplicationError(err error, showError func(string, string)) {
	executable, _ := os.Executable()
	workingDir, _ := os.Getwd()
	log.Printf("event=application_failed pid=%d executable=%q working_directory=%q error=%q", os.Getpid(), executable, workingDir, err)
	fmt.Fprintf(os.Stderr, "Screen OCR LLM failed: %v\n", err)

	logging := "File logging is disabled. Set ENABLE_FILE_LOGGING=true in your configuration to record diagnostics."
	if strings.EqualFold(os.Getenv("ENABLE_FILE_LOGGING"), "true") {
		logging = "Configured diagnostic log: " + filepath.Join(workingDir, "screen_ocr_debug.log")
	}
	message := fmt.Sprintf("%v\n\nExecutable: %s\nWorking directory: %s\n\n%s\n\nPress Ctrl+C in this dialog to copy its details.", err, executable, workingDir, logging)
	showError("Screen OCR LLM - Error", message)
}
