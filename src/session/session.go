package session

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"time"

	"screen-ocr-llm/src/clipboard"
	"screen-ocr-llm/src/ocr"
	"screen-ocr-llm/src/popup"
	"screen-ocr-llm/src/screenshot"
	"screen-ocr-llm/src/singleinstance"
)

var ErrSelectionCancelled = errors.New("selection cancelled")

type RegionSelectorFunc func(ctx context.Context) (screenshot.Region, bool, error)

type RecognizeFunc func(ctx context.Context, region screenshot.Region) (string, error)

type ResultTarget interface {
	OnSuccess(text string) error
	OnFailure(err error) error
}

type PopupController interface {
	StartCountdown(timeoutSeconds int) error
	UpdateText(text string) error
	Close() error
}

type Options struct {
	Deadline               time.Duration
	SelectRegion           RegionSelectorFunc
	Recognize              RecognizeFunc
	Target                 ResultTarget
	Popup                  PopupController
	SuccessVisibleDuration time.Duration
}

type Result struct {
	Text string
}

func Execute(ctx context.Context, opts Options) (Result, error) {
	if opts.SelectRegion == nil {
		return Result{}, errors.New("SelectRegion is required")
	}
	if opts.Target == nil {
		return Result{}, errors.New("Target is required")
	}

	region, cancelled, err := opts.SelectRegion(ctx)
	if err != nil {
		_ = opts.Target.OnFailure(err)
		return Result{}, err
	}
	if cancelled {
		_ = opts.Target.OnFailure(ErrSelectionCancelled)
		return Result{}, ErrSelectionCancelled
	}

	deadline := opts.Deadline
	if deadline <= 0 {
		deadline = 20 * time.Second
	}

	recognize := opts.Recognize
	if recognize == nil {
		recognize = recognizeWithContext
	}

	p := opts.Popup
	if p == nil {
		p = defaultPopupController{}
	}

	countdownSeconds := int(math.Ceil(deadline.Seconds()))
	if countdownSeconds < 1 {
		countdownSeconds = 1
	}
	_ = p.StartCountdown(countdownSeconds)

	jobCtx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	text, err := recognize(jobCtx, region)
	if err != nil {
		_ = p.Close()
		_ = opts.Target.OnFailure(err)
		return Result{}, err
	}

	if err := opts.Target.OnSuccess(text); err != nil {
		_ = p.Close()
		_ = opts.Target.OnFailure(err)
		return Result{}, err
	}

	_ = p.UpdateText(text)

	if opts.SuccessVisibleDuration > 0 {
		time.Sleep(opts.SuccessVisibleDuration)
	}

	return Result{Text: text}, nil
}

type defaultPopupController struct{}

func (defaultPopupController) StartCountdown(timeoutSeconds int) error {
	return popup.StartCountdown(timeoutSeconds)
}

func (defaultPopupController) UpdateText(text string) error {
	return popup.UpdateText(text)
}

func (defaultPopupController) Close() error {
	return popup.Close()
}

type ClipboardTarget struct{}

func (ClipboardTarget) OnSuccess(text string) error {
	return clipboard.Write(text)
}

func (ClipboardTarget) OnFailure(err error) error {
	return nil
}

type StdoutTarget struct {
	Writer io.Writer
}

func (t StdoutTarget) OnSuccess(text string) error {
	w := t.Writer
	if w == nil {
		w = os.Stdout
	}
	_, err := fmt.Fprint(w, text)
	return err
}

func (t StdoutTarget) OnFailure(err error) error {
	return nil
}

type DelegatedTarget struct {
	Conn           singleinstance.Conn
	OutputToStdout bool
}

func (t DelegatedTarget) OnSuccess(text string) error {
	if t.Conn == nil {
		return errors.New("delegated target missing connection")
	}
	if t.OutputToStdout {
		return t.Conn.RespondSuccess(text)
	}
	if err := clipboard.Write(text); err != nil {
		return fmt.Errorf("clipboard error: %w", err)
	}
	return t.Conn.RespondSuccess("")
}

func (t DelegatedTarget) OnFailure(err error) error {
	if t.Conn == nil {
		return nil
	}
	if err == nil {
		return t.Conn.RespondError("unknown session error")
	}
	return t.Conn.RespondError(err.Error())
}

func recognizeWithContext(ctx context.Context, region screenshot.Region) (string, error) {
	resCh := make(chan struct {
		text string
		err  error
	}, 1)

	go func() {
		text, err := ocr.Recognize(region)
		resCh <- struct {
			text string
			err  error
		}{text: text, err: err}
	}()

	select {
	case r := <-resCh:
		return r.text, r.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
