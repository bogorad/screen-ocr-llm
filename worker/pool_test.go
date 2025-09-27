package worker

import (
	"context"
	"testing"
	"time"

	"screen-ocr-llm/screenshot"
)

func TestPoolSubmitDropWhenBusy(t *testing.T) {
	p := New(1)
	defer p.Close()
	ctx := context.Background()
	r := screenshot.Region{X: 0, Y: 0, Width: 1, Height: 1}

	done := make(chan struct{})
	// First submit occupies the single queue slot or worker
	ok := p.Submit(ctx, r, func(string, error) { time.Sleep(100 * time.Millisecond); close(done) })
	if !ok { t.Fatal("first submit should succeed") }
	// Immediately try a second submit; with 1-slot queue, it may still succeed once, but the next should drop
	ok2 := p.Submit(ctx, r, func(string, error) {})
	// Third submit must drop given 1-slot queue and one in-flight
	ok3 := p.Submit(ctx, r, func(string, error) {})
	if ok2 && ok3 { t.Fatal("expected at least one submit to drop due to full queue") }
	<-done
}

