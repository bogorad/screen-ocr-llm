package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"screen-ocr-llm/src/singleinstance"
)

func main() {
	n := flag.Int("n", 50, "number of clients to launch")
	mode := flag.String("mode", "std", "std|clip: run-once-std (stdout) or run-once (clipboard)")
	deadline := flag.Duration("deadline", 5*time.Second, "per-client timeout")
	flag.Parse()

	var wg sync.WaitGroup
	var okCount int32
	var busyCount int32
	var errCount int32

	start := time.Now()
	for i := 0; i < *n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), *deadline)
			defer cancel()
			client := singleinstance.NewClient()
			stdout := *mode == "std"
			delegated, _, err := client.TryRunOnce(ctx, stdout)
			if err != nil {
				// Count busy separately if server responds with Busy
				if strings.Contains(strings.ToLower(err.Error()), "busy") {
					atomic.AddInt32(&busyCount, 1)
					return
				}
				atomic.AddInt32(&errCount, 1)
				return
			}
			if delegated {
				atomic.AddInt32(&okCount, 1)
				return
			}
			// No resident; treat as error for stress purposes
			atomic.AddInt32(&errCount, 1)
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)
	fmt.Fprintf(os.Stdout, "launched=%d ok=%d busy=%d err=%d elapsed=%s\n", *n, okCount, busyCount, errCount, elapsed)
}
