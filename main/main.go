package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"screen-ocr-llm/clipboard"
	"screen-ocr-llm/config"
	"screen-ocr-llm/gui"
	"screen-ocr-llm/hotkey"
	"screen-ocr-llm/llm"
	"screen-ocr-llm/ocr"
	"screen-ocr-llm/screenshot"
	"screen-ocr-llm/tray"
)

func main() {
	// Parse command line flags
	runOnce := flag.Bool("runonce", false, "Run OCR once and exit (no tray icon)")
	flag.Parse()

	// If runonce mode, skip single instance check and run OCR immediately
	if *runOnce {
		runOCROnce()
		return
	}

	// Ensure single instance
	ensureSingleInstance()
	defer os.Remove(pidFile) // Clean up PID file on exit

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
		APIKey: cfg.APIKey,
		Model:  cfg.Model,
	})
	err = clipboard.Init()
	if err != nil {
		log.Fatalf("Failed to initialize clipboard: %v", err)
	}

	log.Printf("Screen OCR LLM Tool initialized")
	log.Printf("Using model: %s", cfg.Model)
	log.Printf("Hotkey: %s", cfg.Hotkey)

	// Create system tray icon
	exitRequested := make(chan bool, 1)
	trayIcon, err := tray.New(tray.Config{
		Title:   "Screen OCR Tool",
		Tooltip: fmt.Sprintf("Screen OCR Tool - Press %s to capture", cfg.Hotkey),
		OnExit: func() {
			log.Printf("Exit requested from tray icon")
			exitRequested <- true
		},
	})
	if err != nil {
		log.Printf("Failed to create tray icon: %v", err)
	} else {
		log.Printf("System tray icon created")
		// Run tray icon in a separate goroutine
		go trayIcon.Run()
		defer trayIcon.Destroy()
	}

	// Start hotkey listener with integrated workflow
	hotkey.Listen(cfg.Hotkey, func() {
		// This callback is no longer used as the workflow is now integrated
		// into the hotkey package itself
		log.Printf("Legacy callback - workflow now integrated in hotkey package")
	})

	// Wait for interrupt signal or tray exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		log.Printf("Shutting down due to signal...")
	case <-exitRequested:
		log.Printf("Shutting down due to tray exit...")
	}
}

func setupLogging(enableFileLogging bool) {
	if enableFileLogging {
		// Create or open log file
		logFile, err := os.OpenFile("screen_ocr_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Printf("Failed to open log file: %v", err)
			return
		}

		// Set up multi-writer to write to both file and stdout
		multiWriter := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(multiWriter)
		log.Printf("File logging enabled: screen_ocr_debug.log")
	} else {
		log.SetOutput(os.Stdout)
	}

	// Set log format with timestamp
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

const pidFile = "screen-ocr-llm.pid"

func ensureSingleInstance() {
	// Check if PID file exists
	if _, err := os.Stat(pidFile); err == nil {
		// Read existing PID
		pidBytes, err := ioutil.ReadFile(pidFile)
		if err == nil {
			if oldPid, err := strconv.Atoi(string(pidBytes)); err == nil {
				// Try to kill the old process
				if process, err := os.FindProcess(oldPid); err == nil {
					log.Printf("Found existing instance with PID %d, killing it...", oldPid)
					process.Kill()
					process.Wait() // Wait for it to die
					log.Printf("Old instance killed")
				}
			}
		}
	}

	// Write current PID to file
	currentPid := os.Getpid()
	pidStr := fmt.Sprintf("%d", currentPid)
	if err := ioutil.WriteFile(pidFile, []byte(pidStr), 0644); err != nil {
		log.Printf("Warning: Could not write PID file: %v", err)
	} else {
		log.Printf("Running as PID %d", currentPid)
	}
}

// runOCROnce performs a single OCR capture and exits
func runOCROnce() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

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
		APIKey: cfg.APIKey,
		Model:  cfg.Model,
	})
	if err := clipboard.Init(); err != nil {
		log.Fatalf("Failed to initialize clipboard: %v", err)
	}

	log.Printf("Running OCR once (--runonce mode)")

	// Set up a completion channel
	done := make(chan bool, 1)
	var extractedText string

	// Set up the region selection callback
	gui.SetRegionSelectionCallback(func(region screenshot.Region) error {
		log.Printf("Processing region: %+v", region)

		// Perform OCR on the selected region
		text, err := ocr.Recognize(region)
		if err != nil {
			log.Printf("OCR failed: %v", err)
			done <- true
			return err
		}

		log.Printf("OCR extracted text (%d chars): %q", len(text), text)

		// Copy result to clipboard
		if err := clipboard.Write(text); err != nil {
			log.Printf("Failed to write to clipboard: %v", err)
			done <- true
			return err
		}

		extractedText = text
		log.Printf("OCR completed successfully, text copied to clipboard (%d chars)", len(text))
		done <- true
		return nil
	})

	// Start region selection
	if err := gui.StartRegionSelection(); err != nil {
		log.Fatalf("Failed to start region selection: %v", err)
	}

	// Wait for completion
	<-done

	fmt.Printf("Extracted text: %s\n", extractedText)
}
