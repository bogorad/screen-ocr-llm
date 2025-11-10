package eventloop

import (
	"context"
	"fmt"
	"log"
	"time"

	"screen-ocr-llm/src/clipboard"
	"screen-ocr-llm/src/config"
	"screen-ocr-llm/src/hotkey"
	"screen-ocr-llm/src/overlay"
	"screen-ocr-llm/src/popup"
	"screen-ocr-llm/src/screenshot"
	"screen-ocr-llm/src/singleinstance"
	"screen-ocr-llm/src/tray"
	"screen-ocr-llm/src/worker"
)

// Loop is the single-threaded coordinator for IPC-based run-once and hotkey flows.
type Loop struct {
	selector       overlay.Selector
	pool           *worker.Pool
	srv            singleinstance.Server
	busy           bool
	results        chan result
	hotkeyCh       chan struct{}
	defaultTooltip string
	deadline       time.Duration
}

type result struct {
	text   string
	err    error
	conn   singleinstance.Conn
	stdout bool
	cancel context.CancelFunc
}

// New creates a new event loop with defaults based on config.
// If cfg is nil or cfg.OCRDeadlineSec <= 0, a 20s deadline is used.
func New(cfg *config.Config) *Loop {
	deadlineSec := 20
	if cfg != nil && cfg.OCRDeadlineSec > 0 {
		deadlineSec = cfg.OCRDeadlineSec
	}

	return &Loop{
		selector:       overlay.NewSelector(),
		pool:           worker.New(0),
		results:        make(chan result, 1),
		hotkeyCh:       make(chan struct{}, 4),
		defaultTooltip: "Screen OCR Tool",
		deadline:       time.Duration(deadlineSec) * time.Second,
	}
}

// SetDefaultTooltip optionally sets the tray tooltip base text.
func (l *Loop) SetDefaultTooltip(tt string) { l.defaultTooltip = tt }

func (l *Loop) setBusy(b bool) {
	l.busy = b
	if b {
		tray.UpdateTooltip("Screen OCR: processing...")
	} else {
		tray.UpdateTooltip(l.defaultTooltip)
	}
}

// StartHotkey registers a global hotkey and posts events into the loop.
func (l *Loop) StartHotkey(combo string) {
	if combo == "" {
		return
	}
	hotkey.Listen(combo, func() {
		select {
		case l.hotkeyCh <- struct{}{}:
		default:
		}
	})
}

// Run starts the singleinstance server and processes client requests.
// It blocks until ctx is cancelled.
func (l *Loop) Run(ctx context.Context) error {
	l.srv = singleinstance.NewServer()
	if err := l.srv.Start(ctx); err != nil {
		return err
	}
	// Update tray About with port info
	if p := l.srv.Port(); p > 0 {
		log.Printf("Resident listening on 127.0.0.1:%d", p)
		tray.SetAboutExtra(fmt.Sprintf("Resident TCP port: %d", p))
	}
	defer l.pool.Close()

	// Accept loop in background to avoid blocking result handling
	reqCh := make(chan singleinstance.Conn, 4)
	go func() {
		for {
			conn, err := l.srv.Next(ctx)
			if err != nil {
				close(reqCh)
				return
			}
			reqCh <- conn
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-l.hotkeyCh:
			l.handleHotkey(ctx)
		case conn, ok := <-reqCh:
			if !ok {
				return nil
			}
			l.handleConn(ctx, conn)
		case res := <-l.results:
			l.handleResult(res)
		}
	}
}

func (l *Loop) handleConn(ctx context.Context, conn singleinstance.Conn) {
	if l.busy {
		_ = conn.RespondError("Busy, please retry")
		_ = conn.Close()
		return
	}

	req := conn.Request()
	region, cancelled, err := l.selectRegion(ctx)
	if err != nil {
		_ = conn.RespondError("Failed to select region: " + err.Error())
		_ = conn.Close()
		return
	}
	if cancelled {
		_ = conn.RespondError("Selection cancelled")
		_ = conn.Close()
		return
	}

	jobCtx, cancel := context.WithTimeout(ctx, l.deadline)

	// Start countdown popup for delegated --run-once requests
	_ = popup.StartCountdown(int(l.deadline.Seconds()))

	l.setBusy(true)
	submitted := l.pool.Submit(jobCtx, region, func(text string, err error) {
		l.results <- result{text: text, err: err, conn: conn, stdout: req.OutputToStdout, cancel: cancel}
	})
	if !submitted {
		cancel() // Clean up context if job wasn't submitted
		l.setBusy(false)
		_ = popup.Close()
		_ = conn.RespondError("Busy, please retry")
		_ = conn.Close()
		return
	}
}

func (l *Loop) handleResult(res result) {
	log.Printf("handleResult: called with text length=%d, err=%v, conn=%v", len(res.text), res.err, res.conn != nil)
	defer func() {
		l.setBusy(false)
		if res.cancel != nil {
			res.cancel()
		}
	}()
	// Pipe-client path
	if res.conn != nil {
		defer res.conn.Close()
		if res.err != nil {
			_ = popup.Close() // Close countdown popup on error
			_ = res.conn.RespondError(res.err.Error())
			return
		}
		if res.stdout {
			_ = popup.UpdateText(res.text) // Update countdown to result
			_ = res.conn.RespondSuccess(res.text)
			return
		}
		if err := clipboard.Write(res.text); err != nil {
			_ = popup.Close() // Close countdown popup on error
			_ = res.conn.RespondError("Clipboard error: " + err.Error())
			return
		}
		_ = popup.UpdateText(res.text) // Update countdown to result
		_ = res.conn.RespondSuccess("")
		return
	}
	// Resident hotkey path
	log.Printf("handleResult: resident hotkey path")
	if res.err != nil {
		// Timeout or error - just close popup silently
		log.Printf("handleResult: error in resident path: %v", res.err)
		_ = popup.Close()
		return
	}
	log.Printf("handleResult: writing %d chars to clipboard", len(res.text))
	if err := clipboard.Write(res.text); err != nil {
		log.Printf("handleResult: clipboard error: %v", err)
		_ = popup.Close()
		_ = popup.Show("Clipboard error")
		return
	}
	// Update countdown popup with result text
	log.Printf("handleResult: updating popup with result")
	_ = popup.UpdateText(res.text)
	log.Printf("handleResult: hotkey flow complete")
}

func (l *Loop) handleHotkey(ctx context.Context) {
	log.Printf("handleHotkey: called")
	if l.busy {
		log.Printf("handleHotkey: busy, skipping")
		_ = popup.Show("Busy, please retry")
		return
	}
	log.Printf("handleHotkey: selecting region")
	region, cancelled, err := l.selectRegion(ctx)
	if err != nil {
		log.Printf("handleHotkey: selection error: %v", err)
		_ = popup.Show("Selection error")
		return
	}
	if cancelled {
		log.Printf("handleHotkey: selection cancelled")
		return
	}

	log.Printf("handleHotkey: region selected %dx%d, creating job context with deadline %v", region.Width, region.Height, l.deadline)
	jobCtx, cancel := context.WithTimeout(ctx, l.deadline)

	// Start countdown popup immediately
	log.Printf("handleHotkey: starting countdown popup")
	_ = popup.StartCountdown(int(l.deadline.Seconds()))

	l.setBusy(true)
	log.Printf("handleHotkey: submitting job to worker pool")
	ok := l.pool.Submit(jobCtx, region, func(text string, err error) {
		log.Printf("handleHotkey: callback invoked with text length=%d, err=%v", len(text), err)
		l.results <- result{text: text, err: err, conn: nil, stdout: false, cancel: cancel}
	})
	if !ok {
		log.Printf("handleHotkey: submit failed, pool busy")
		cancel() // Clean up context if job wasn't submitted
		l.setBusy(false)
		_ = popup.Close()
		_ = popup.Show("Busy, please retry")
	} else {
		log.Printf("handleHotkey: job submitted successfully")
	}
}

func (l *Loop) selectRegion(ctx context.Context) (screenshot.Region, bool, error) {
	return l.selector.Select(ctx)
}

// Deadline returns the configured OCR deadline for this loop.
func (l *Loop) Deadline() time.Duration { return l.deadline }
