package eventloop

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"screen-ocr-llm/src/config"
	"screen-ocr-llm/src/hotkey"
	"screen-ocr-llm/src/overlay"
	"screen-ocr-llm/src/popup"
	"screen-ocr-llm/src/screenshot"
	"screen-ocr-llm/src/session"
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
	target resultTarget
	cancel context.CancelFunc
}

type resultTarget interface {
	OnSuccess(text string) error
	OnProcessError(err error)
	OnDeliveryError(err error)
	Close()
}

type hotkeyResultTarget struct{}

func (hotkeyResultTarget) OnSuccess(text string) error {
	return session.ClipboardTarget{}.OnSuccess(text)
}

func (hotkeyResultTarget) OnProcessError(err error) {}

func (hotkeyResultTarget) OnDeliveryError(err error) {
	_ = popup.Show("Clipboard error")
}

func (hotkeyResultTarget) Close() {}

type delegatedResultTarget struct {
	sink session.DelegatedTarget
	conn singleinstance.Conn
}

func newDelegatedResultTarget(conn singleinstance.Conn, outputToStdout bool) delegatedResultTarget {
	return delegatedResultTarget{
		sink: session.DelegatedTarget{Conn: conn, OutputToStdout: outputToStdout},
		conn: conn,
	}
}

func (t delegatedResultTarget) OnSuccess(text string) error {
	return t.sink.OnSuccess(text)
}

func (t delegatedResultTarget) OnProcessError(err error) {
	_ = t.sink.OnFailure(err)
}

func (t delegatedResultTarget) OnDeliveryError(err error) {
	_ = t.sink.OnFailure(err)
}

func (t delegatedResultTarget) Close() {
	if t.conn != nil {
		_ = t.conn.Close()
	}
}

type requestCallbacks struct {
	onBusy        func()
	onSelectError func(err error)
	onCancelled   func()
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
	target := newDelegatedResultTarget(conn, conn.Request().OutputToStdout)
	l.startRequest(ctx, target, requestCallbacks{
		onBusy: func() {
			target.OnProcessError(errors.New("Busy, please retry"))
			target.Close()
		},
		onSelectError: func(err error) {
			target.OnProcessError(fmt.Errorf("Failed to select region: %w", err))
			target.Close()
		},
		onCancelled: func() {
			target.OnProcessError(session.ErrSelectionCancelled)
			target.Close()
		},
	})
}

func (l *Loop) handleResult(res result) {
	log.Printf("handleResult: called with text length=%d, err=%v", len(res.text), res.err)
	defer func() {
		l.setBusy(false)
		if res.cancel != nil {
			res.cancel()
		}
	}()
	if res.target == nil {
		log.Printf("handleResult: missing target")
		_ = popup.Close()
		return
	}
	defer res.target.Close()

	if res.err != nil {
		log.Printf("handleResult: processing error: %v", res.err)
		_ = popup.Close()
		res.target.OnProcessError(res.err)
		return
	}

	if err := res.target.OnSuccess(res.text); err != nil {
		log.Printf("handleResult: delivery error: %v", err)
		_ = popup.Close()
		res.target.OnDeliveryError(err)
		return
	}

	// Update countdown popup with result text
	log.Printf("handleResult: updating popup with result")
	_ = popup.UpdateText(res.text)
}

func (l *Loop) handleHotkey(ctx context.Context) {
	log.Printf("handleHotkey: called")
	l.startRequest(ctx, hotkeyResultTarget{}, requestCallbacks{
		onBusy: func() {
			log.Printf("handleHotkey: busy, skipping")
			_ = popup.Show("Busy, please retry")
		},
		onSelectError: func(err error) {
			log.Printf("handleHotkey: selection error: %v", err)
			_ = popup.Show("Selection error")
		},
		onCancelled: func() {
			log.Printf("handleHotkey: selection cancelled")
		},
	})
}

func (l *Loop) startRequest(ctx context.Context, target resultTarget, callbacks requestCallbacks) {
	if l.busy {
		if callbacks.onBusy != nil {
			callbacks.onBusy()
		}
		return
	}

	region, cancelled, err := l.selectRegion(ctx)
	if err != nil {
		if callbacks.onSelectError != nil {
			callbacks.onSelectError(err)
		}
		return
	}
	if cancelled {
		if callbacks.onCancelled != nil {
			callbacks.onCancelled()
		}
		return
	}

	jobCtx, cancel := context.WithTimeout(ctx, l.deadline)
	_ = popup.StartCountdown(int(l.deadline.Seconds()))

	l.setBusy(true)
	submitted := l.pool.Submit(jobCtx, region, func(text string, err error) {
		l.results <- result{text: text, err: err, target: target, cancel: cancel}
	})
	if !submitted {
		cancel()
		l.setBusy(false)
		_ = popup.Close()
		if callbacks.onBusy != nil {
			callbacks.onBusy()
		}
	}
}

func (l *Loop) selectRegion(ctx context.Context) (screenshot.Region, bool, error) {
	return l.selector.Select(ctx)
}

// Deadline returns the configured OCR deadline for this loop.
func (l *Loop) Deadline() time.Duration { return l.deadline }
