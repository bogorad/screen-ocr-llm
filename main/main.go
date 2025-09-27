package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"screen-ocr-llm/clipboard"
	"screen-ocr-llm/config"
	"screen-ocr-llm/eventloop"
	"screen-ocr-llm/gui"
	"screen-ocr-llm/llm"
	"screen-ocr-llm/logutil"
	"screen-ocr-llm/ocr"
	"screen-ocr-llm/popup"
	"screen-ocr-llm/screenshot"
	"screen-ocr-llm/singleinstance"
	"screen-ocr-llm/tray"
)

// normalizeFlagDashes maps GNU-style --run-once[(-std)] to Go's -run-once[(-std)]
func normalizeFlagDashes() {
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch {
		case arg == "--run-once":
			os.Args[i] = "-run-once"
		case strings.HasPrefix(arg, "--run-once="):
			os.Args[i] = "-run-once" + arg[len("--run-once"):]
		}
	}
}


func main() {
	// Parse command line flags
	runOnce := flag.Bool("run-once", false, "Run OCR once, copy to clipboard, and exit silently")
	// Support GNU-style double-dash flags
	normalizeFlagDashes()

	flag.Parse()

	// If run-once mode, prefer delegating to resident via TCP; fallback to standalone
	if *runOnce {
		stdout := false
		ctx := context.Background()
		client := singleinstance.NewClient()
		delegated, _, err := client.TryRunOnce(ctx, stdout)
		if err != nil {
			log.Printf("Delegation error: %v; falling back to standalone", err)
			runOCROnce(stdout)
			return
		}
		if delegated {
			log.Printf("Delegated to resident")
			return
		}
		log.Printf("No resident detected (not delegated), running standalone")
		// Fallback to standalone
		runOCROnce(stdout)
		return
	}

	// Pre-flight: set up file logging early so detection logs are captured
	logutil.Setup(true)
	// Load .env early so SINGLEINSTANCE_PORT_* are available for pre-flight
	_, _ = config.Load()
	// ---------- SINGLE-INSTANCE NUKE ----------
	startPort, _ := singleinstance.GetPortRangeForDebug()
	addr := fmt.Sprintf("127.0.0.1:%d", startPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("Pre-flight: port %d busy → resident already exists", startPort)
		fmt.Printf("one is already running on port %d\n", startPort)
		os.Exit(1)
	}
	// We claimed the port; release it so the event loop can re-bind.
	_ = listener.Close()
	log.Printf("Pre-flight: port %d free → we are the one true resident", startPort)
	// ------------------------------------------

	// Named-pipe single instance enforced by event loop server; PID file removed

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logging
	setupLogging(cfg.EnableFileLogging)

	// Validate configuration
	if cfg.APIKey == "" {
		log.Fatalf("OPENROUTER_API_KEY is required. Please set it in your .env file.")
	}
	if cfg.Model == "" {
		log.Fatalf("MODEL is required. Please set it in your .env file.")
	}

	// Initialize packages
	screenshot.Init()
	ocr.Init()
	llm.Init(&llm.Config{
		APIKey:    cfg.APIKey,
		Model:     cfg.Model,
		Providers: cfg.Providers,
	})
	err = clipboard.Init()
	if err != nil {
		log.Fatalf("Failed to initialize clipboard: %v", err)
	}

	log.Printf("Screen OCR LLM Tool initialized")
	log.Printf("Using model: %s", cfg.Model)
	log.Printf("Hotkey: %s", cfg.Hotkey)

	// Event loop + tray + hotkey
	loop := eventloop.New()
	loop.SetDefaultTooltip(fmt.Sprintf("Screen OCR Tool - Press %s to capture", cfg.Hotkey))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	trayIcon, _ := tray.New(tray.Config{
		Title:   "Screen OCR Tool",
		Tooltip: fmt.Sprintf("Screen OCR Tool - Press %s to capture", cfg.Hotkey),
		OnExit:  func() { cancel() },
	})
	go trayIcon.Run()
	defer trayIcon.Destroy()

	loop.StartHotkey(cfg.Hotkey)

	// Handle SIGINT/SIGTERM
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		cancel()
	}()

	if err := loop.Run(ctx); err != nil {
		log.Printf("event loop stopped: %v", err)
	}


}

func setupLogging(enableFileLogging bool) {
	logutil.Setup(enableFileLogging)
}


// runOCROnce performs a single OCR capture and exits
func runOCROnce(outputToStdout bool) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Set up logging to file only (no stdout output)
	setupLogging(true) // Force file logging

	// Validate configuration
	if cfg.APIKey == "" {
		fmt.Fprintf(os.Stderr, "OPENROUTER_API_KEY is required. Please set it in your .env file.\n")
		os.Exit(1)
	}
	if cfg.Model == "" {
		fmt.Fprintf(os.Stderr, "MODEL is required. Please set it in your .env file.\n")
		os.Exit(1)
	}

	// Initialize packages
	screenshot.Init()
	ocr.Init()
	llm.Init(&llm.Config{
		APIKey:    cfg.APIKey,
		Model:     cfg.Model,
		Providers: cfg.Providers,
	})

	// Always initialize clipboard for consistent behavior
	// (even if we won't use it in stdout mode)
	if err := clipboard.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize clipboard: %v\n", err)
		os.Exit(1)
	}

	log.Printf("Running OCR once (--runonce mode)")

	// Set up completion channels
	done := make(chan bool, 1)
	errorChan := make(chan error, 1)

	// Set up the region selection callback
	gui.SetRegionSelectionCallback(func(region screenshot.Region) error {
		log.Printf("Processing region: %+v", region)

		// Perform OCR on the selected region
		text, err := ocr.Recognize(region)
		if err != nil {
			log.Printf("OCR failed: %v", err)
			errorChan <- fmt.Errorf("OCR failed: %v", err)
			return nil // Don't return error to avoid confusing with region selection failure
		}

		// Log OCR result safely (prevent log injection)
		safeText := sanitizeForLogging(text)
		log.Printf("OCR extracted text (%d chars): %q", len(text), safeText)

		if outputToStdout {
			// Output to stdout for --run-once-std mode
			fmt.Print(text) // Use Print (not Println) to avoid extra newline
			log.Printf("OCR completed successfully, text output to stdout (%d chars)", len(text))
		} else {
			// Copy result to clipboard for --run-once mode
			if err := clipboard.Write(text); err != nil {
				log.Printf("Failed to write to clipboard: %v", err)
				errorChan <- fmt.Errorf("Failed to write to clipboard: %v", err)
				return nil // Don't return error to avoid confusing with region selection failure
			}
			log.Printf("OCR completed successfully, text copied to clipboard (%d chars)", len(text))
		}
		// Always show popup for visibility in standalone run-once
		log.Printf("Standalone: requesting popup")
		_ = popup.Show(text)
		// Block long enough for the popup to be visible before process exit
		time.Sleep(3 * time.Second)
		log.Printf("Sending completion signal...")
		done <- true
		log.Printf("Completion signal sent")
		return nil
	})

	// Start region selection
	if err := gui.StartRegionSelection(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start region selection: %v\n", err)
		os.Exit(1)
	}

	// Wait for completion or error
	log.Printf("Waiting for OCR completion...")
	select {
	case <-done:
		log.Printf("OCR completion signal received")
		log.Printf("OCR runonce completed successfully, exiting...")
		os.Exit(0)
	case err := <-errorChan:
		log.Printf("OCR process failed: %v", err)
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

// sanitizeForLogging removes potentially dangerous characters from text for safe logging
func sanitizeForLogging(text string) string {
	// Limit length to prevent log flooding
	const maxLogLength = 100
	if len(text) > maxLogLength {
		text = text[:maxLogLength] + "..."
	}

	// Replace newlines and other control characters to prevent log injection
	sanitized := ""
	for _, r := range text {
		if r == '\n' || r == '\r' {
			sanitized += "\\n"
		} else if r == '\t' {
			sanitized += "\\t"
		} else if r < 32 || r == 127 { // Control characters
			sanitized += "?"
		} else {
			sanitized += string(r)
		}
	}
	return sanitized
}
