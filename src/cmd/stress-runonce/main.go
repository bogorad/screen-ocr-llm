package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"

	"screen-ocr-llm/src/singleinstance"
)

type stressOptions struct {
	n        int
	mode     string
	deadline time.Duration
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	opts := &stressOptions{}
	cmd := newRootCmd(opts)
	return cmd.Execute()
}

func newRootCmd(opts *stressOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "stress-runonce",
		Short:         "Stress test run-once delegation",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithOptions(*opts)
		},
	}

	cmd.Flags().IntVar(&opts.n, "n", 50, "number of clients to launch")
	cmd.Flags().StringVar(&opts.mode, "mode", "std", "std|clip: run-once-std (stdout) or run-once (clipboard)")
	cmd.Flags().DurationVar(&opts.deadline, "deadline", 5*time.Second, "per-client timeout")

	return cmd
}

func runWithOptions(opts stressOptions) error {
	var wg sync.WaitGroup
	var okCount int32
	var busyCount int32
	var errCount int32

	start := time.Now()
	for i := 0; i < opts.n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), opts.deadline)
			defer cancel()
			client := singleinstance.NewClient()
			stdout := opts.mode == "std"
			delegated, _, err := client.TryRunOnce(ctx, stdout)
			if err != nil {
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
			atomic.AddInt32(&errCount, 1)
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)
	fmt.Fprintf(os.Stdout, "launched=%d ok=%d busy=%d err=%d elapsed=%s\n", opts.n, okCount, busyCount, errCount, elapsed)
	return nil
}
