package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"screen-ocr-llm/src/clipboard"
	"screen-ocr-llm/src/config"
	"screen-ocr-llm/src/eventloop"
	"screen-ocr-llm/src/gui"
	"screen-ocr-llm/src/llm"
	"screen-ocr-llm/src/logutil"
	"screen-ocr-llm/src/notification"

	"screen-ocr-llm/src/ocr"
	"screen-ocr-llm/src/popup"
	"screen-ocr-llm/src/screenshot"
	"screen-ocr-llm/src/singleinstance"
	"screen-ocr-llm/src/tray"
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

// enableDPIAwareness attempts to set per-monitor DPI awareness on Windows to fix scaling issues
func enableDPIAwareness() {
	if runtime.GOOS != "windows" {
		return
	}
	// Prefer per-monitor DPI awareness via Shcore.SetProcessDpiAwareness (Win 8.1+)
	shcore := syscall.NewLazyDLL("Shcore.dll")
	setProcessDpiAwareness := shcore.NewProc("SetProcessDpiAwareness")
	const PROCESS_PER_MONITOR_DPI_AWARE = 2
	if err := setProcessDpiAwareness.Find(); err == nil {
		_, _, _ = setProcessDpiAwareness.Call(uintptr(PROCESS_PER_MONITOR_DPI_AWARE))
		return
	}
	// Fallback: user32.SetProcessDPIAware (Vista+)
	user32 := syscall.NewLazyDLL("user32.dll")
	setProcessDPIAware := user32.NewProc("SetProcessDPIAware")
	if err := setProcessDPIAware.Find(); err == nil {
		_, _, _ = setProcessDPIAware.Call()
	}
}

func main() {
	// Ensure DPI awareness before creating any windows or querying metrics
	enableDPIAwareness()

	// Lock main goroutine to its own OS thread to prevent it from sharing
	// the popup thread's message queue
	runtime.LockOSThread()

	// Parse command line flags
	runOnce := flag.Bool("run-once", false, "Run OCR once, copy to clipboard, and exit silently")
	// Support GNU-style double-dash flags
	normalizeFlagDashes()

	flag.Parse()

	// If run-once mode, prefer delegating to resident via TCP; fallback to standalone
	if *runOnce {
		// Load .env early so SINGLEINSTANCE_PORT_* are applied before delegation scan
		_, _ = config.Load()
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

	// Initialize LLM first and validate immediately (blocking dialog on failure)
	llm.Init(&llm.Config{
		APIKey:    cfg.APIKey,
		Model:     cfg.Model,
		Providers: cfg.Providers,
	})
	if err := llm.Ping(); err != nil {
		notification.ShowBlockingError("LLM unavailable", fmt.Sprintf("Startup check failed: %v\n\nPlease verify your API key and network connectivity.", err))
		os.Exit(1)
	}
	log.Printf("LLM ping succeeded")

	// Initialize remaining packages
	screenshot.Init()
	ocr.Init()
	err = clipboard.Init()
	if err != nil {
		log.Fatalf("Failed to initialize clipboard: %v", err)
	}

	log.Printf("Screen OCR LLM Tool initialized")
	log.Printf("Using model: %s", cfg.Model)
	log.Printf("Hotkey: %s", cfg.Hotkey)
	log.Printf("OCR deadline: %ds", cfg.OCRDeadlineSec)

	// Propagate hotkey to About dialog
	tray.SetAboutHotkey(cfg.Hotkey)

	// Event loop + tray + hotkey
	loop := eventloop.New(cfg)
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

	// Set up logging per configuration (file logging may be disabled)
	setupLogging(cfg.EnableFileLogging)

	// Validate configuration
	if cfg.APIKey == "" {
		fmt.Fprintf(os.Stderr, "OPENROUTER_API_KEY is required. Please set it in your .env file.\n")
		os.Exit(1)
	}
	if cfg.Model == "" {
		fmt.Fprintf(os.Stderr, "MODEL is required. Please set it in your .env file.\n")
		os.Exit(1)
	}

	// Initialize LLM first and validate immediately (blocking dialog on failure)
	llm.Init(&llm.Config{
		APIKey:    cfg.APIKey,
		Model:     cfg.Model,
		Providers: cfg.Providers,
	})
	if err := llm.Ping(); err != nil {
		notification.ShowBlockingError("LLM unavailable", fmt.Sprintf("Startup check failed: %v\n\nPlease verify your API key and network connectivity.", err))
		os.Exit(1)
	}
	log.Printf("LLM ping succeeded")

	// Initialize remaining packages
	screenshot.Init()
	ocr.Init()

	// Always initialize clipboard for consistent behavior
	// (even if we won't use it in stdout mode)
	if err := clipboard.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize clipboard: %v\n", err)
		os.Exit(1)
	}

	log.Printf("Running OCR once (--runonce mode) with OCR deadline %ds", cfg.OCRDeadlineSec)

	// Start region selection
	region, err := gui.StartRegionSelection()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start region selection: %v\n", err)
		os.Exit(1)
	}

	log.Printf("Processing region: %+v", region)

	// Start countdown popup before OCR using configured deadline
	_ = popup.StartCountdown(cfg.OCRDeadlineSec)

	// Perform OCR on the selected region
	text, err := ocr.Recognize(region)
	if err != nil {
		_ = popup.Close() // Close countdown on error
		log.Printf("OCR failed: %v", err)
		fmt.Fprintf(os.Stderr, "OCR failed: %v\n", err)
		os.Exit(1)
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
			_ = popup.Close()
			log.Printf("Failed to write to clipboard: %v", err)
			fmt.Fprintf(os.Stderr, "Failed to write to clipboard: %v\n", err)
			os.Exit(1)
		}
		log.Printf("OCR completed successfully, text copied to clipboard (%d chars)", len(text))
	}

	// Update countdown popup with result
	_ = popup.UpdateText(text)
	// Block long enough for the popup to be visible before process exit
	time.Sleep(3 * time.Second)

	log.Printf("OCR runonce completed successfully, exiting...")
	os.Exit(0)
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
