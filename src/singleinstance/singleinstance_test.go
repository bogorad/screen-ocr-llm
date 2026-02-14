package singleinstance

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestServerClientRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv := NewServer()
	if err := srv.Start(ctx); err != nil {
		t.Skipf("named pipe unavailable in this environment: %v", err)
	}
	defer srv.Close()

	// client delegates stdout request
	client := NewClient()
	errCh := make(chan error, 1)
	go func() {
		delegated, _, err := client.TryRunOnce(ctx, true)
		if err != nil {
			errCh <- fmt.Errorf("client: %w", err)
			return
		}
		if !delegated {
			errCh <- fmt.Errorf("expected delegation")
			return
		}
		errCh <- nil
	}()

	// server accept and respond
	conn, err := srv.Next(ctx)
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if !conn.Request().OutputToStdout {
		t.Errorf("expected stdout request")
	}
	if err := conn.RespondSuccess("ok"); err != nil {
		t.Fatalf("respond: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatalf("client did not complete: %v", ctx.Err())
	}
}
