package main

import (
	"context"
	"errors"
	"testing"
)

func TestNormalizeLegacyArgs(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		out  []string
	}{
		{
			name: "Normalizes long single dash flags",
			in:   []string{"screen-ocr-llm", "-run-once", "-api-key-path", "/tmp/key"},
			out:  []string{"screen-ocr-llm", "--run-once", "--api-key-path", "/tmp/key"},
		},
		{
			name: "Normalizes equals form",
			in:   []string{"screen-ocr-llm", "-run-once=true", "-api-key-path=/tmp/key"},
			out:  []string{"screen-ocr-llm", "--run-once=true", "--api-key-path=/tmp/key"},
		},
		{
			name: "Leaves other flags unchanged",
			in:   []string{"screen-ocr-llm", "--run-once", "--other"},
			out:  []string{"screen-ocr-llm", "--run-once", "--other"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeLegacyArgs(tt.in)
			if len(got) != len(tt.out) {
				t.Fatalf("Expected len=%d, got %d", len(tt.out), len(got))
			}
			for i := range got {
				if got[i] != tt.out[i] {
					t.Fatalf("Expected arg[%d]=%q, got %q", i, tt.out[i], got[i])
				}
			}
		})
	}
}

func TestNewRootCmdParsesFlags(t *testing.T) {
	opts := &mainOptions{}
	cmd := newRootCmd(opts)
	if err := cmd.ParseFlags([]string{"--run-once", "--api-key-path", "/tmp/key"}); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}
	if !opts.runOnce {
		t.Fatal("Expected runOnce=true")
	}
	if opts.apiKeyPath != "/tmp/key" {
		t.Fatalf("Expected apiKeyPath=/tmp/key, got %q", opts.apiKeyPath)
	}
}

type fakeClient struct {
	delegated bool
	err       error
	called    bool
}

func (f *fakeClient) TryRunOnce(ctx context.Context, outputToStdout bool) (bool, string, error) {
	f.called = true
	return f.delegated, "", f.err
}

func TestHandleRunOnceWithDelegation_Delegated(t *testing.T) {
	client := &fakeClient{delegated: true}
	fallbackCalled := false

	handleRunOnceWithDelegation("", client, func() {
		fallbackCalled = true
	})

	if !client.called {
		t.Fatal("Expected client.TryRunOnce to be called")
	}
	if fallbackCalled {
		t.Fatal("Did not expect fallback when delegation succeeds")
	}
}

func TestHandleRunOnceWithDelegation_NoResidentFallback(t *testing.T) {
	client := &fakeClient{delegated: false}
	fallbackCalled := false

	handleRunOnceWithDelegation("", client, func() {
		fallbackCalled = true
	})

	if !client.called {
		t.Fatal("Expected client.TryRunOnce to be called")
	}
	if !fallbackCalled {
		t.Fatal("Expected fallback when no resident is delegated")
	}
}

func TestHandleRunOnceWithDelegation_DelegationErrorFallback(t *testing.T) {
	client := &fakeClient{err: errors.New("busy")}
	fallbackCalled := false

	handleRunOnceWithDelegation("", client, func() {
		fallbackCalled = true
	})

	if !client.called {
		t.Fatal("Expected client.TryRunOnce to be called")
	}
	if !fallbackCalled {
		t.Fatal("Expected fallback when delegation returns an error")
	}
}
