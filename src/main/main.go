package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"screen-ocr-llm/src/config"
	"screen-ocr-llm/src/eventloop"
	"screen-ocr-llm/src/logutil"
	"screen-ocr-llm/src/overlay"
	"screen-ocr-llm/src/runtimeinit"
	"screen-ocr-llm/src/screenshot"
	"screen-ocr-llm/src/session"
	"screen-ocr-llm/src/singleinstance"
	"screen-ocr-llm/src/tray"
)

type mainOptions struct {
	runOnce    bool
	apiKeyPath string
}

func normalizeLegacyArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}

	normalized := make([]string, len(args))
	copy(normalized, args)

	for i := 1; i < len(normalized); i++ {
		arg := normalized[i]
		switch {
		case arg == "-run-once":
			normalized[i] = "--run-once"
		case strings.HasPrefix(arg, "-run-once="):
			normalized[i] = "--run-once=" + arg[len("-run-once="):]
		case arg == "-api-key-path":
			normalized[i] = "--api-key-path"
		case strings.HasPrefix(arg, "-api-key-path="):
			normalized[i] = "--api-key-path=" + arg[len("-api-key-path="):]
		}
	}

	return normalized
}

func run() error {
	return runWithArgs(normalizeLegacyArgs(os.Args))
}

func runWithArgs(args []string) error {
	if len(args) == 0 {
		args = []string{"screen-ocr-llm"}
	}

	opts := &mainOptions{}
	cmd := newRootCmd(opts)
	cmd.SetArgs(args[1:])
	return cmd.Execute()
}

func newRootCmd(opts *mainOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "screen-ocr-llm",
		Short:         "Screen OCR LLM resident app",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runApplication(*opts)
		},
	}

	cmd.Flags().BoolVar(&opts.runOnce, "run-once", false, "Run OCR once, copy to clipboard, and exit silently")
	cmd.Flags().StringVar(&opts.apiKeyPath, "api-key-path", "", "Path to API key file (highest precedence)")

	return cmd
}

func main() {
	if err := run(); err != nil {
		log.Printf("Application failed: %v", err)
		os.Exit(1)
	}
}

func runApplication(opts mainOptions) error {
	// Ensure DPI awareness before creating any windows or querying metrics
	enableDPIAwareness()
	logMonitorConfiguration()

	// Lock main goroutine to its own OS thread to prevent it from sharing
	// the popup thread's message queue
	runtime.LockOSThread()

	// If run-once mode, prefer delegating to resident via TCP; fallback to standalone
	if opts.runOnce {
		handleRunOnceWithDelegation(opts.apiKeyPath, singleinstance.NewClient(), func() {
			runOCROnce(false, opts.apiKeyPath)
		})
		return nil
	}

	// Load .env early so SINGLEINSTANCE_PORT_* are available for pre-flight
	_, _ = config.LoadWithOptions(config.LoadOptions{APIKeyPathOverride: opts.apiKeyPath})
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

	cfg, err := runtimeinit.Bootstrap(runtimeinit.Options{
		LoadOptions:          config.LoadOptions{APIKeyPathOverride: opts.apiKeyPath},
		SetupLogging:         setupLogging,
		ShowBlockingLLMError: true,
	})
	if err != nil {
		return err
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

	return nil
}

func setupLogging(enableFileLogging bool) {
	logutil.Setup(enableFileLogging)
}

// runOCROnce performs a single OCR capture and exits
func runOCROnce(outputToStdout bool, apiKeyPathOverride string) {
	cfg, err := runtimeinit.Bootstrap(runtimeinit.Options{
		LoadOptions:          config.LoadOptions{APIKeyPathOverride: apiKeyPathOverride},
		SetupLogging:         setupLogging,
		ShowBlockingLLMError: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize runtime: %v\n", err)
		os.Exit(1)
	}

	log.Printf("Running OCR once (--runonce mode) with OCR deadline %ds", cfg.OCRDeadlineSec)

	selector := overlay.NewSelector()
	var target session.ResultTarget
	if outputToStdout {
		target = session.StdoutTarget{Writer: os.Stdout}
	} else {
		target = runOnceClipboardTarget{}
	}

	_, err = session.Execute(context.Background(), session.Options{
		Deadline: time.Duration(cfg.OCRDeadlineSec) * time.Second,
		SelectRegion: func(ctx context.Context) (screenshot.Region, bool, error) {
			region, cancelled, err := selector.Select(ctx)
			if err != nil {
				return screenshot.Region{}, false, fmt.Errorf("failed to start region selection: %w", err)
			}
			return region, cancelled, nil
		},
		Target:                 target,
		SuccessVisibleDuration: 3 * time.Second,
	})
	if err != nil {
		switch {
		case errors.Is(err, session.ErrSelectionCancelled):
			fmt.Fprintf(os.Stderr, "Selection cancelled\n")
		case isClipboardWriteError(err):
			fmt.Fprintf(os.Stderr, "Failed to write to clipboard: %v\n", err)
		case isRegionSelectionError(err):
			fmt.Fprintf(os.Stderr, "Failed to start region selection: %v\n", err)
		default:
			fmt.Fprintf(os.Stderr, "OCR failed: %v\n", err)
		}
		os.Exit(1)
	}

	log.Printf("OCR runonce completed successfully, exiting...")
	os.Exit(0)
}

func handleRunOnceWithDelegation(apiKeyPathOverride string, client singleinstance.Client, runFallback func()) {
	// Load .env early so SINGLEINSTANCE_PORT_* are applied before delegation scan.
	_, _ = config.LoadWithOptions(config.LoadOptions{APIKeyPathOverride: apiKeyPathOverride})

	delegated, _, err := client.TryRunOnce(context.Background(), false)
	if err != nil {
		log.Printf("Delegation error: %v; falling back to standalone", err)
		runFallback()
		return
	}
	if delegated {
		log.Printf("Delegated to resident")
		return
	}

	log.Printf("No resident detected (not delegated), running standalone")
	runFallback()
}

type runOnceClipboardTarget struct{}

func (runOnceClipboardTarget) OnSuccess(text string) error {
	if err := (session.ClipboardTarget{}).OnSuccess(text); err != nil {
		return fmt.Errorf("clipboard write: %w", err)
	}
	return nil
}

func (runOnceClipboardTarget) OnFailure(err error) error { return nil }

func isClipboardWriteError(err error) bool {
	return strings.Contains(err.Error(), "clipboard write")
}

func isRegionSelectionError(err error) bool {
	return strings.Contains(err.Error(), "failed to start region selection")
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
