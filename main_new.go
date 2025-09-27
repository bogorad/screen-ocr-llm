//go:build legacy

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"screen-ocr-llm/config"
	"screen-ocr-llm/llm"
	"screen-ocr-llm/messages"
	"screen-ocr-llm/process"
	"screen-ocr-llm/router"
	"screen-ocr-llm/screenshot"
	"screen-ocr-llm/singleinstance"
	"screen-ocr-llm/tray"

	clipboardProcess "screen-ocr-llm/processes/clipboard"
	configProcess "screen-ocr-llm/processes/config"
	hotkeyProcess "screen-ocr-llm/processes/hotkey"
	ocrProcess "screen-ocr-llm/processes/ocr"
	popupProcess "screen-ocr-llm/processes/popup"
	regionProcess "screen-ocr-llm/processes/region"
	trayProcess "screen-ocr-llm/processes/tray"

	winio "github.com/Microsoft/go-winio"
)

func main() {
	// Parse command line flags
	runOnce := flag.Bool("run-once", false, "Run OCR once, copy to clipboard, and exit silently")
	flag.Parse()

	// If run-once mode, check for resident and delegate or fallback
	if *runOnce {
		runOnceWithDelegation(false) // false = copy to clipboard, silent
		return
	}


	// Ensure single instance
	ensureSingleInstance()
	defer os.Remove(pidFile)

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

	// Initialize core packages
	screenshot.Init()
	llm.Init(&llm.Config{
		APIKey:    cfg.APIKey,
		Model:     cfg.Model,
		Providers: cfg.Providers,
	})

	log.Printf("Screen OCR LLM Tool initialized with new parallel architecture")
	log.Printf("Using model: %s", cfg.Model)
	log.Printf("Hotkey: %s", cfg.Hotkey)

	// Create process manager
	manager := process.NewManager()

	// Register all processes
	if err := registerProcesses(manager, cfg); err != nil {
		log.Fatalf("Failed to register processes: %v", err)
	}

	// Start all processes
	if err := manager.StartAll(); err != nil {
		log.Fatalf("Failed to start processes: %v", err)
	}

	log.Printf("All processes started successfully")

	// Start main coordinator
	coordinator := NewCoordinator(manager, cfg)
	go coordinator.Run()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down...", sig)
	case <-coordinator.Done():
		log.Printf("Coordinator finished, shutting down...")
	}

	// Graceful shutdown
	log.Printf("Initiating graceful shutdown...")
	coordinator.Stop()
	manager.StopAll()
	log.Printf("Shutdown complete")
}

// registerProcesses registers all application processes with the manager
func registerProcesses(manager *process.Manager, cfg *config.Config) error {
	processes := []process.Process{
		hotkeyProcess.NewProcess(cfg.Hotkey),
		regionProcess.NewProcess(),
		ocrProcess.NewProcess(),
		clipboardProcess.NewProcess(),
		popupProcess.NewProcess(),
		trayProcess.NewProcess(tray.Config{
			Title:   "Screen OCR Tool",
			Tooltip: fmt.Sprintf("Screen OCR Tool - Press %s to capture", cfg.Hotkey),
		}),
		configProcess.NewProcess(),
	}

	for _, proc := range processes {
		if err := manager.Register(proc); err != nil {
			return fmt.Errorf("failed to register process %s: %v", proc.Name(), err)
		}
	}

	return nil
}

// Coordinator manages the main application workflow
type Coordinator struct {
	manager *process.Manager
	router  *router.Router
	config  *config.Config
	ctx     context.Context
	cancel  context.CancelFunc
	done    chan struct{}

	// Run-once delegation state
	runOnceMode         bool
	runOnceOutputToStdout bool
	runOnceResponseFile string
}

// NewCoordinator creates a new main coordinator
func NewCoordinator(manager *process.Manager, cfg *config.Config) *Coordinator {
	ctx, cancel := context.WithCancel(context.Background())
	return &Coordinator{
		manager: manager,
		router:  manager.GetRouter(),
		config:  cfg,
		ctx:     ctx,
		cancel:  cancel,
		done:    make(chan struct{}),
	}
}

// Run starts the coordinator main loop
func (c *Coordinator) Run() {
	defer close(c.done)

	// Register main process with router
	channel, err := c.router.RegisterProcess(messages.ProcessMain, 50)
	if err != nil {
		log.Printf("Failed to register main process: %v", err)
		return
	}

	log.Printf("Main coordinator started")

	// Start file-based request monitoring
	go c.monitorRunOnceRequests()

	// Main message handling loop
	for {
		select {
		case envelope := <-channel:
			c.handleMessage(envelope)
		case <-c.ctx.Done():
			log.Printf("Main coordinator stopping")
			return
		}
	}
}

// Stop gracefully stops the coordinator
func (c *Coordinator) Stop() {
	c.cancel()
}

// Done returns a channel that closes when the coordinator is done
func (c *Coordinator) Done() <-chan struct{} {
	return c.done
}

// handleMessage processes incoming messages
func (c *Coordinator) handleMessage(envelope messages.MessageEnvelope) {
	switch msg := envelope.Message.(type) {
	case messages.HotkeyPressed:
		log.Printf("Main: Hotkey pressed: %s", msg.Combo)
		c.handleHotkeyPressed(msg)
	case messages.RegionSelected:
		log.Printf("Main: Region selected: %+v", msg.Region)
		c.handleRegionSelected(msg)
	case messages.RegionCancelled:
		log.Printf("Main: Region selection cancelled")
		c.handleRegionCancelled()
	case messages.OCRComplete:
		log.Printf("Main: OCR complete")
		c.handleOCRComplete(msg)
	case messages.ClipboardComplete:
		log.Printf("Main: Clipboard operation complete")
		c.handleClipboardComplete(msg)
	case messages.TrayMenuClicked:
		log.Printf("Main: Tray menu clicked: %s", msg.Action)
		c.handleTrayMenuClicked(msg)
	case messages.ConfigChanged:
		log.Printf("Main: Configuration changed")
		c.handleConfigChanged(msg)
	default:
		log.Printf("Main: Received unknown message type: %s from %s", msg.Type(), envelope.From)
	}
}

// handleHotkeyPressed starts the OCR workflow
func (c *Coordinator) handleHotkeyPressed(msg messages.HotkeyPressed) {
	log.Printf("Main: Starting OCR workflow for hotkey: %s", msg.Combo)

	// Start region selection
	err := c.router.Send(messages.MessageEnvelope{
		From:    messages.ProcessMain,
		To:      messages.ProcessRegionSel,
		Message: messages.StartRegionSelection{},
	})

	if err != nil {
		log.Printf("Main: Failed to send region selection request: %v", err)
	}
}

// handleRegionSelected processes successful region selection
func (c *Coordinator) handleRegionSelected(msg messages.RegionSelected) {
	log.Printf("Main: Processing selected region: %+v", msg.Region)

	// Send region to OCR process
	err := c.router.Send(messages.MessageEnvelope{
		From:    messages.ProcessMain,
		To:      messages.ProcessOCR,
		Message: messages.ProcessRegion{Region: msg.Region},
	})

	if err != nil {
		log.Printf("Main: Failed to send OCR request: %v", err)
	}
}

// handleRegionCancelled processes cancelled region selection
func (c *Coordinator) handleRegionCancelled() {
	log.Printf("Main: Region selection was cancelled, OCR workflow aborted")
}

// handleOCRComplete processes OCR completion
func (c *Coordinator) handleOCRComplete(msg messages.OCRComplete) {
	if msg.Error != nil {
		log.Printf("Main: OCR failed: %v", msg.Error)
		if c.runOnceMode {
			c.sendRunOnceResponse("", msg.Error)
		}
		return
	}

	log.Printf("Main: OCR successful, extracted %d characters", len(msg.Text))

	// Handle run-once mode
	if c.runOnceMode {
		if c.runOnceOutputToStdout {
			// For --run-once-std, just send response immediately
			c.sendRunOnceResponse(msg.Text, nil)
			return
		} else {
			// For --run-once, copy to clipboard first, then send response
			err := c.router.Send(messages.MessageEnvelope{
				From:    messages.ProcessMain,
				To:      messages.ProcessClipboard,
				Message: messages.WriteClipboard{Text: msg.Text},
			})

			if err != nil {
				log.Printf("Main: Failed to send clipboard request: %v", err)
				c.sendRunOnceResponse("", err)
				return
			}
			// Response will be sent in handleClipboardComplete
			return
		}
	}

	// Normal resident mode - send to clipboard and popup
	err := c.router.Send(messages.MessageEnvelope{
		From:    messages.ProcessMain,
		To:      messages.ProcessClipboard,
		Message: messages.WriteClipboard{Text: msg.Text},
	})

	if err != nil {
		log.Printf("Main: Failed to send clipboard request: %v", err)
		return
	}

	// Send text to popup (fire and forget)
	err = c.router.Send(messages.MessageEnvelope{
		From:    messages.ProcessMain,
		To:      messages.ProcessPopup,
		Message: messages.ShowPopup{Text: msg.Text, Duration: 3},
	})

	if err != nil {
		log.Printf("Main: Failed to send popup request: %v", err)
	}
}

// handleClipboardComplete processes clipboard operation completion
func (c *Coordinator) handleClipboardComplete(msg messages.ClipboardComplete) {
	if msg.Error != nil {
		log.Printf("Main: Clipboard operation failed: %v", msg.Error)
		if c.runOnceMode {
			c.sendRunOnceResponse("", msg.Error)
		}
	} else {
		log.Printf("Main: OCR workflow completed successfully")
		if c.runOnceMode {
			// For --run-once mode, send success response after clipboard is written
			c.sendRunOnceResponse("", nil) // Empty text since it's already in clipboard
		}
	}
}

// handleTrayMenuClicked processes tray menu interactions
func (c *Coordinator) handleTrayMenuClicked(msg messages.TrayMenuClicked) {
	switch msg.Action {
	case "exit":
		log.Printf("Main: Exit requested from tray menu")
		c.Stop()
	case "about":
		log.Printf("Main: About requested from tray menu")
		// Could show about dialog
	default:
		log.Printf("Main: Unknown tray action: %s", msg.Action)
	}
}

// handleConfigChanged processes configuration changes
func (c *Coordinator) handleConfigChanged(msg messages.ConfigChanged) {
	log.Printf("Main: Configuration changed, updating application")

	// Update internal config
	c.config = &msg.Config

	// Reinitialize LLM with new config
	llm.Init(&llm.Config{
		APIKey:    msg.Config.APIKey,
		Model:     msg.Config.Model,
		Providers: msg.Config.Providers,
	})

	// Update tray tooltip
	err := c.router.Send(messages.MessageEnvelope{
		From: messages.ProcessMain,
		To:   messages.ProcessTray,
		Message: messages.UpdateTray{
			Tooltip: fmt.Sprintf("Screen OCR Tool - Press %s to capture", msg.Config.Hotkey),
			Status:  "updated",
		},
	})

	if err != nil {
		log.Printf("Main: Failed to update tray: %v", err)
	}

	log.Printf("Main: Configuration update completed")
}

// monitorRunOnceRequests monitors for file-based run-once requests and DIENOW signals
func (c *Coordinator) monitorRunOnceRequests() {
	requestFile := "runonce_request.tmp"
	dienowFile := "dienow_signal.tmp"

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			// Check for DIENOW signal file first
			if _, err := os.Stat(dienowFile); err == nil {
				log.Printf("Main: Detected DIENOW signal file, shutting down...")
				os.Remove(dienowFile) // Clean up the signal file
				c.Stop()
				return
			}

			// Check for run-once request file
			if _, err := os.Stat(requestFile); err == nil {
				log.Printf("Main: Detected run-once request file")
				c.handleRunOnceRequest(requestFile)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// handleRunOnceRequest processes a file-based run-once request
func (c *Coordinator) handleRunOnceRequest(requestFile string) {
	// Read request
	data, err := os.ReadFile(requestFile)
	if err != nil {
		log.Printf("Main: Failed to read request file: %v", err)
		return
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) < 2 {
		log.Printf("Main: Invalid request format")
		return
	}

	outputToStdout := lines[0] == "true"
	responseFile := lines[1]

	log.Printf("Main: Processing run-once request (stdout=%v)", outputToStdout)

	// Remove request file immediately to prevent reprocessing
	os.Remove(requestFile)

	// Start OCR workflow
	c.startRunOnceWorkflow(outputToStdout, responseFile)
}

// startRunOnceWorkflow starts the OCR workflow for run-once requests
func (c *Coordinator) startRunOnceWorkflow(outputToStdout bool, responseFile string) {
	// Store run-once state in coordinator
	c.runOnceMode = true
	c.runOnceOutputToStdout = outputToStdout
	c.runOnceResponseFile = responseFile

	log.Printf("Main: Starting run-once OCR workflow")

	// Send region selection request
	err := c.router.Send(messages.MessageEnvelope{
		From:    messages.ProcessMain,
		To:      messages.ProcessRegionSel,
		Message: messages.StartRegionSelection{},
	})

	if err != nil {
		log.Printf("Main: Failed to send region selection request: %v", err)
		c.sendRunOnceResponse("", fmt.Errorf("failed to start region selection: %v", err))
		return
	}

	log.Printf("Main: Sent region selection request for run-once")
}

// sendRunOnceResponse sends the response back to the requesting process
func (c *Coordinator) sendRunOnceResponse(text string, err error) {
	if c.runOnceResponseFile == "" {
		return
	}

	var response string
	if err != nil {
		response = fmt.Sprintf("ERROR\n%v", err)
	} else {
		response = fmt.Sprintf("SUCCESS\n%s", text)
	}

	if writeErr := os.WriteFile(c.runOnceResponseFile, []byte(response), 0644); writeErr != nil {
		log.Printf("Main: Failed to write response file: %v", writeErr)
	} else {
		log.Printf("Main: Sent run-once response (%d chars)", len(text))
	}

	// Reset run-once state
	c.runOnceMode = false
	c.runOnceOutputToStdout = false
	c.runOnceResponseFile = ""
}

// Utility functions from original main.go

const pidFile = "screen-ocr-llm.pid"

func ensureSingleInstance() {
	currentPid := os.Getpid()

	// Try to create PID file exclusively to prevent symlink attacks
	f, err := os.OpenFile(pidFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		// File exists, check if process is still running
		if os.IsExist(err) {
			// Read existing PID safely
			existingFile, readErr := os.Open(pidFile)
			if readErr == nil {
				pidBytes := make([]byte, 32) // Reasonable limit for PID
				n, readErr := existingFile.Read(pidBytes)
				existingFile.Close()

				if readErr == nil && n > 0 {
					if oldPid, parseErr := strconv.Atoi(string(pidBytes[:n])); parseErr == nil {
						// Check if process is still running
						if process, findErr := os.FindProcess(oldPid); findErr == nil {
							// Try to signal the process (non-destructive check)
							if signalErr := process.Signal(syscall.Signal(0)); signalErr == nil {
								log.Printf("Found existing instance with PID %d, attempting graceful shutdown...", oldPid)

								// Step 1: Send DIENOW message via file
								if err := sendDienowToExistingProcess(); err != nil {
									log.Printf("Failed to send DIENOW message: %v", err)
								} else {
									log.Printf("Sent DIENOW message to existing process")
								}

								// Step 2: Wait 5 seconds for graceful shutdown
								log.Printf("Waiting 5 seconds for graceful shutdown...")
								for i := 0; i < 50; i++ { // 5 seconds, check every 100ms
									time.Sleep(100 * time.Millisecond)
									if process.Signal(syscall.Signal(0)) != nil {
										// Process is dead
										log.Printf("Existing process shut down gracefully")
										break
									}
								}

								// Step 3: Check if still running, force kill if needed
								if process.Signal(syscall.Signal(0)) == nil {
									log.Printf("Process still running after 5 seconds, force killing...")
									killCmd := fmt.Sprintf("taskkill /F /PID %d", oldPid)
									if err := exec.Command("cmd", "/C", killCmd).Run(); err != nil {
										log.Printf("Failed to kill process with taskkill: %v", err)
										// Fallback to Go's Kill method
										process.Kill()
									}
									log.Printf("Force killed existing process")
								}
							}
						}
					}
				}
			}

			// Remove stale PID file and try again
			os.Remove(pidFile)
			f, err = os.OpenFile(pidFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		}

		if err != nil {
			log.Fatalf("Failed to create PID file (single-instance check failed): %v", err)
		}
	}

	// Write current PID to file
	pidStr := fmt.Sprintf("%d", currentPid)
	if _, writeErr := f.WriteString(pidStr); writeErr != nil {
		f.Close()
		log.Fatalf("Failed to write PID to file: %v", writeErr)
	}
	f.Close()

	log.Printf("Running as PID %d", currentPid)
}

// sendDienowToExistingProcess sends a DIENOW message to existing process via file
func sendDienowToExistingProcess() error {
	dienowFile := "dienow_signal.tmp"

	// Create DIENOW signal file
	if err := os.WriteFile(dienowFile, []byte("DIENOW"), 0644); err != nil {
		return fmt.Errorf("failed to create DIENOW file: %v", err)
	}

	// The existing process should detect this file and shut down
	// We don't remove the file here - let the existing process clean it up
	return nil
}

func setupLogging(enableFileLogging bool) {
	if enableFileLogging {
		// Create or open log file
		logFile, err := os.OpenFile("screen_ocr_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
			return
		}

		// ALWAYS write only to file (keep stdout clean)
		log.SetOutput(logFile)
		log.Printf("File logging enabled: screen_ocr_debug.log")
	} else {
		// If file logging is disabled, discard logs (keep stdout clean)
		log.SetOutput(io.Discard)
	}

	// Set log format with timestamp
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

// runOnceWithDelegation checks for resident process and delegates or falls back
func runOnceWithDelegation(outputToStdout bool) {
	// Check if resident process is running
	if isResidentRunning() {
		log.Printf("Resident process detected, delegating run-once request")
		if err := delegateToResident(outputToStdout); err != nil {
			log.Printf("Delegation failed: %v, falling back to standalone mode", err)
			runOCROnce(outputToStdout)
		}
	} else {
		log.Printf("No resident process detected, running standalone")
		runOCROnce(outputToStdout)
	}
}

// isResidentRunning checks for the named pipe presence by trying a quick connect
func isResidentRunning() bool {
	d := 200 * time.Millisecond
	if _, err := winio.DialPipe(singleinstance.PipeName, &d); err == nil {
		return true
	}
	return false
}

// delegateToResident sends a run-once request to the resident process via named pipe
func delegateToResident(outputToStdout bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := singleinstance.NewClient()
	delegated, text, err := client.TryRunOnce(ctx, outputToStdout)
	if err != nil { return err }
	if !delegated { return fmt.Errorf("no resident available") }
	if outputToStdout { fmt.Print(text) }
	return nil
}

// runOCROnce performs a single OCR capture and exits using full resident architecture
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

	// Initialize core packages
	screenshot.Init()
	llm.Init(&llm.Config{
		APIKey:    cfg.APIKey,
		Model:     cfg.Model,
		Providers: cfg.Providers,
	})

	log.Printf("Running OCR once using full resident architecture (outputToStdout=%v)", outputToStdout)

	// Use the SAME process manager and architecture as resident mode
	runOnceWithFullArchitecture(cfg, outputToStdout)
}

// runOnceWithFullArchitecture runs OCR once using the same architecture as resident mode
func runOnceWithFullArchitecture(cfg *config.Config, outputToStdout bool) {
	// Create process manager (same as resident mode)
	manager := process.NewManager()

	// Register processes - SKIP hotkey, tray, and config for run-once
	processes := []process.Process{
		regionProcess.NewProcess(),
		ocrProcess.NewProcess(),
		popupProcess.NewProcess(),
	}

	// Add clipboard process only if not outputting to stdout
	if !outputToStdout {
		processes = append(processes, clipboardProcess.NewProcess())
	}

	for _, proc := range processes {
		if err := manager.Register(proc); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to register process %s: %v\n", proc.Name(), err)
			os.Exit(1)
		}
	}

	// Start all processes
	if err := manager.StartAll(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start processes: %v\n", err)
		os.Exit(1)
	}

	log.Printf("Full architecture started for run-once mode")

	// Create coordinator (same as resident mode but with run-once behavior)
	coordinator := NewRunOnceCoordinator(manager, cfg, outputToStdout)

	// Run the coordinator - this will immediately start OCR workflow and exit when done
	coordinator.Run()

	// Cleanup
	log.Printf("Run-once completed, shutting down processes")
	manager.StopAll()
}

// RunOnceCoordinator manages the run-once workflow using the same architecture as resident mode
type RunOnceCoordinator struct {
	manager       *process.Manager
	router        *router.Router
	config        *config.Config
	outputToStdout bool
	done          chan struct{}
	result        chan bool
}

// NewRunOnceCoordinator creates a new run-once coordinator
func NewRunOnceCoordinator(manager *process.Manager, cfg *config.Config, outputToStdout bool) *RunOnceCoordinator {
	return &RunOnceCoordinator{
		manager:       manager,
		router:        manager.GetRouter(),
		config:        cfg,
		outputToStdout: outputToStdout,
		done:          make(chan struct{}),
		result:        make(chan bool, 1),
	}
}

// Run executes the run-once workflow using the same message passing as resident mode
func (roc *RunOnceCoordinator) Run() {
	// Register main process with router
	channel, err := roc.router.RegisterProcess(messages.ProcessMain, 20)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register main process: %v\n", err)
		os.Exit(1)
	}

	log.Printf("Run-once coordinator started, immediately beginning OCR workflow")

	// Immediately start the OCR workflow (no waiting for hotkeys)
	err = roc.router.Send(messages.MessageEnvelope{
		From:    messages.ProcessMain,
		To:      messages.ProcessRegionSel,
		Message: messages.StartRegionSelection{},
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start region selection: %v\n", err)
		os.Exit(1)
	}

	// Message handling loop (same as resident mode)
	go func() {
		defer close(roc.done)

		for {
			select {
			case envelope := <-channel:
				if roc.handleMessage(envelope) {
					return // Workflow completed
				}
			}
		}
	}()

	// Wait for completion
	<-roc.done

	// Wait for final result
	select {
	case success := <-roc.result:
		if success {
			log.Printf("Run-once coordinator: OCR workflow completed successfully")
			os.Exit(0)
		} else {
			log.Printf("Run-once coordinator: OCR workflow failed")
			os.Exit(1)
		}
	default:
		log.Printf("Run-once coordinator: OCR workflow completed")
		os.Exit(0)
	}
}

// handleMessage processes messages using the same logic as resident coordinator
func (roc *RunOnceCoordinator) handleMessage(envelope messages.MessageEnvelope) bool {
	switch msg := envelope.Message.(type) {
	case messages.RegionSelected:
		log.Printf("Run-once coordinator: Region selected: %+v", msg.Region)
		return roc.handleRegionSelected(msg)

	case messages.RegionCancelled:
		log.Printf("Run-once coordinator: Region selection cancelled")
		roc.result <- false
		return true

	case messages.OCRComplete:
		log.Printf("Run-once coordinator: OCR complete")
		return roc.handleOCRComplete(msg)

	case messages.ClipboardComplete:
		log.Printf("Run-once coordinator: Clipboard operation complete")
		return roc.handleClipboardComplete(msg)

	default:
		log.Printf("Run-once coordinator: Received unknown message type: %s from %s", msg.Type(), envelope.From)
		return false
	}
}

// handleRegionSelected processes successful region selection (same as resident coordinator)
func (roc *RunOnceCoordinator) handleRegionSelected(msg messages.RegionSelected) bool {
	log.Printf("Run-once coordinator: Processing selected region: %+v", msg.Region)

	// Send region to OCR process (same as resident mode)
	err := roc.router.Send(messages.MessageEnvelope{
		From:    messages.ProcessMain,
		To:      messages.ProcessOCR,
		Message: messages.ProcessRegion{Region: msg.Region},
	})

	if err != nil {
		log.Printf("Run-once coordinator: Failed to send OCR request: %v", err)
		roc.result <- false
		return true
	}

	return false // Continue processing
}

// handleOCRComplete processes OCR completion (same logic as resident coordinator)
func (roc *RunOnceCoordinator) handleOCRComplete(msg messages.OCRComplete) bool {
	if msg.Error != nil {
		log.Printf("Run-once coordinator: OCR failed: %v", msg.Error)
		fmt.Fprintf(os.Stderr, "OCR failed: %v\n", msg.Error)
		roc.result <- false
		return true
	}

	log.Printf("Run-once coordinator: OCR successful, extracted %d characters", len(msg.Text))

	// Show popup notification (same as resident mode)
	err := roc.router.Send(messages.MessageEnvelope{
		From:    messages.ProcessMain,
		To:      messages.ProcessPopup,
		Message: messages.ShowPopup{Text: msg.Text, Duration: 3},
	})

	if err != nil {
		log.Printf("Run-once coordinator: Failed to send popup request: %v", err)
	}

	if roc.outputToStdout {
		// Output to stdout for --run-once-std mode
		fmt.Print(msg.Text) // Use Print (not Println) to avoid extra newline
		log.Printf("Run-once coordinator: OCR completed successfully, text output to stdout (%d chars)", len(msg.Text))
		roc.result <- true
		return true
	} else {
		// Send text to clipboard for --run-once mode (same as resident mode)
		err := roc.router.Send(messages.MessageEnvelope{
			From:    messages.ProcessMain,
			To:      messages.ProcessClipboard,
			Message: messages.WriteClipboard{Text: msg.Text},
		})

		if err != nil {
			log.Printf("Run-once coordinator: Failed to send clipboard request: %v", err)
			roc.result <- false
			return true
		}

		return false // Wait for clipboard completion
	}
}

// handleClipboardComplete processes clipboard operation completion (same as resident coordinator)
func (roc *RunOnceCoordinator) handleClipboardComplete(msg messages.ClipboardComplete) bool {
	if msg.Error != nil {
		log.Printf("Run-once coordinator: Clipboard operation failed: %v", msg.Error)
		fmt.Fprintf(os.Stderr, "Clipboard operation failed: %v\n", msg.Error)
		roc.result <- false
	} else {
		log.Printf("Run-once coordinator: OCR workflow completed successfully")
		roc.result <- true
	}
	return true // Workflow completed
}

// sanitizeForLogging safely formats text for logging to prevent log injection
func sanitizeForLogging(text string) string {
	// Replace newlines and control characters
	safe := strings.ReplaceAll(text, "\n", "\\n")
	safe = strings.ReplaceAll(safe, "\r", "\\r")
	safe = strings.ReplaceAll(safe, "\t", "\\t")

	// Truncate if too long
	if len(safe) > 100 {
		safe = safe[:100] + "..."
	}

	return safe
}
