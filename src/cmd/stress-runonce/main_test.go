package main

import (
	"testing"
	"time"
)

func TestNewRootCmdDefaults(t *testing.T) {
	opts := &stressOptions{}
	cmd := newRootCmd(opts)
	if err := cmd.ParseFlags([]string{}); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}
	if opts.n != 50 {
		t.Fatalf("Expected default n=50, got %d", opts.n)
	}
	if opts.mode != "std" {
		t.Fatalf("Expected default mode=std, got %q", opts.mode)
	}
	if opts.deadline != 5*time.Second {
		t.Fatalf("Expected default deadline=5s, got %v", opts.deadline)
	}
}

func TestNewRootCmdCustomFlags(t *testing.T) {
	opts := &stressOptions{}
	cmd := newRootCmd(opts)
	if err := cmd.ParseFlags([]string{"--n", "3", "--mode", "clip", "--deadline", "7s"}); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}
	if opts.n != 3 {
		t.Fatalf("Expected n=3, got %d", opts.n)
	}
	if opts.mode != "clip" {
		t.Fatalf("Expected mode=clip, got %q", opts.mode)
	}
	if opts.deadline != 7*time.Second {
		t.Fatalf("Expected deadline=7s, got %v", opts.deadline)
	}
}
