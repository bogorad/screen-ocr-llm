package eventloop

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"screen-ocr-llm/clipboard"
	"screen-ocr-llm/hotkey"
	"screen-ocr-llm/overlay"
	"screen-ocr-llm/popup"
	"screen-ocr-llm/screenshot"
	"screen-ocr-llm/singleinstance"
	"screen-ocr-llm/tray"
	"screen-ocr-llm/worker"
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
}

type result struct {
	text   string
	err    error
	conn   singleinstance.Conn
	stdout bool
}

// New creates a new event loop with defaults.
func New() *Loop {
	return &Loop{
		selector:       overlay.NewSelector(),
		pool:           worker.New(0),
		results:        make(chan result, 1),
		hotkeyCh:       make(chan struct{}, 4),
		defaultTooltip: "Screen OCR Tool",
	}
}

// SetDefaultTooltip optionally sets the tray tooltip base text.
func (l *Loop) SetDefaultTooltip(tt string) { l.defaultTooltip = tt }

func (l *Loop) setBusy(b bool) {
    l.busy = b
    if b { tray.UpdateTooltip("Screen OCR: processing...") } else { tray.UpdateTooltip(l.defaultTooltip) }
}

// StartHotkey registers a global hotkey and posts events into the loop.
func (l *Loop) StartHotkey(combo string) {
    if combo == "" { return }
    hotkey.Listen(combo, func() {
        select { case l.hotkeyCh <- struct{}{}: default: }
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

	deadline := ReadDeadline()
	jobCtx, _ := context.WithTimeout(ctx, deadline)

	// Start countdown popup for delegated --run-once requests
	timeoutSeconds := int(deadline.Seconds())
	_ = popup.StartCountdown(timeoutSeconds)

	l.setBusy(true)
	submitted := l.pool.Submit(jobCtx, region, func(text string, err error) {
		l.results <- result{text: text, err: err, conn: conn, stdout: req.OutputToStdout}
	})
	if !submitted {
		l.setBusy(false)
		_ = popup.Close()
		_ = conn.RespondError("Busy, please retry")
		_ = conn.Close()
		return
	}
}

func (l *Loop) handleResult(res result) {
	defer func() { l.setBusy(false) }()
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
	if res.err != nil {
		// Timeout or error - just close popup silently
		_ = popup.Close()
		return
	}
	if err := clipboard.Write(res.text); err != nil {
		_ = popup.Close()
		_ = popup.Show("Clipboard error")
		return
	}
	// Update countdown popup with result text
	_ = popup.UpdateText(res.text)
}
func (l *Loop) handleHotkey(ctx context.Context) {
	if l.busy {
		_ = popup.Show("Busy, please retry")
		return
	}
	region, cancelled, err := l.selectRegion(ctx)
	if err != nil {
		_ = popup.Show("Selection error")
		return
	}
	if cancelled {
		return
	}
	deadline := ReadDeadline()
	jobCtx, _ := context.WithTimeout(ctx, deadline)

	// Start countdown popup immediately
	timeoutSeconds := int(deadline.Seconds())
	_ = popup.StartCountdown(timeoutSeconds)

	l.busy = true
	ok := l.pool.Submit(jobCtx, region, func(text string, err error) {
		l.results <- result{text: text, err: err, conn: nil, stdout: false}
	})
	if !ok {
		l.busy = false
		_ = popup.Close()
		_ = popup.Show("Busy, please retry")
	}
}


func (l *Loop) selectRegion(ctx context.Context) (screenshot.Region, bool, error) {
	return l.selector.Select(ctx)
}

func ReadDeadline() time.Duration {
	v := os.Getenv("OCR_DEADLINE_SEC")
	if v == "" {
		return 15 * time.Second
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		log.Printf("Invalid OCR_DEADLINE_SEC=%q, using default 15s", v)
		return 15 * time.Second
	}
	return time.Duration(n) * time.Second
}

