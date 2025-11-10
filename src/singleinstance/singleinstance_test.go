package singleinstance

import (
	"context"
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
	delegatedCh := make(chan struct{})
	go func() {
		defer close(delegatedCh)
		delegated, _, err := client.TryRunOnce(ctx, true)
		if err != nil {
			t.Errorf("client: %v", err)
		}
		if !delegated {
			t.Errorf("expected delegation")
		}
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
	<-delegatedCh
}
