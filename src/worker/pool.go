package worker

import (
	"context"
	"log"
	"runtime"
	"sync"

	"screen-ocr-llm/src/ocr"
	"screen-ocr-llm/src/screenshot"
)

// ResultCallback is invoked on OCR completion (from a worker goroutine).
// The event loop should pass a closure that posts back into the event loop safely.
type ResultCallback func(text string, err error)

// Pool is a fixed-size OCR worker pool with a 1-slot input queue (strict back-pressure).
type Pool struct {
	jobs chan job
	wg   sync.WaitGroup
}

type job struct {
	ctx    context.Context
	region screenshot.Region
	cb     ResultCallback
}

// New creates a worker pool. Size defaults to NumCPU when size<=0. Queue is 1 slot.
func New(size int) *Pool {
	if size <= 0 {
		size = runtime.NumCPU()
	}
	p := &Pool{jobs: make(chan job, 1)}
	p.start(size)
	return p
}

func (p *Pool) start(n int) {
	for i := 0; i < n; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for j := range p.jobs {
				log.Printf("Worker: Starting OCR for region %dx%d", j.region.Width, j.region.Height)
				// Run OCR with ctx deadline honored inside RecognizeWithContext (to be added)
				text, err := recognizeWithContext(j.ctx, j.region)
				log.Printf("Worker: OCR completed, text length=%d, err=%v", len(text), err)
				log.Printf("Worker: Invoking callback with text length=%d", len(text))
				j.cb(text, err)
				log.Printf("Worker: Callback returned")
			}
		}()
	}
}

// Submit enqueues an OCR job if the single-slot queue is free. Returns false if dropped.
func (p *Pool) Submit(ctx context.Context, region screenshot.Region, cb ResultCallback) bool {
	select {
	case p.jobs <- job{ctx: ctx, region: region, cb: cb}:
		return true
	default:
		return false
	}
}

// Close stops the pool after draining current work.
func (p *Pool) Close() {
	close(p.jobs)
	p.wg.Wait()
}

// recognizeWithContext wraps ocr.Recognize with a deadline-aware path.
func recognizeWithContext(ctx context.Context, region screenshot.Region) (string, error) {
	// Fast path: if no deadline, call existing Recognize.
	if _, ok := ctx.Deadline(); !ok {
		return ocr.Recognize(region)
	}
	// Deadline-aware shim: run in a sub-goroutine, respect ctx.Done().
	// This preserves worker cancellation without touching ocr package yet.
	resCh := make(chan struct {
		text string
		err  error
	}, 1)
	go func() {
		text, err := ocr.Recognize(region)
		resCh <- struct {
			text string
			err  error
		}{text, err}
	}()
	select {
	case r := <-resCh:
		return r.text, r.err
	case <-ctx.Done():
		// Allow underlying OCR to continue in background; we return timeout.
		return "", ctx.Err()
	}
}
