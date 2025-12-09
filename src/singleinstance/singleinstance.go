package singleinstance

// This file defines the API for single-instance ownership and run-once delegation.

import (
	"context"
)

// Server owns the TCP endpoint and answers run-once requests.
type Server interface {
	// Start begins listening on first available port in [49500,49550] and accepting client requests.
	Start(ctx context.Context) error
	// Port returns the bound TCP port, or 0 if not started.
	Port() int
	// Next returns the next accepted connection as a Conn, or ctx error.
	Next(ctx context.Context) (Conn, error)
	// Close releases ownership and stops accepting clients.
	Close() error
}

// Conn represents one client connection and exposes request + response API.
type Conn interface {
	// Request returns the parsed client request.
	Request() Request
	// RespondSuccess sends success. For stdout mode, send text; for clipboard mode, send empty text.
	RespondSuccess(text string) error
	// RespondError sends an error with human-readable message.
	RespondError(msg string) error
	// Close closes the underlying connection.
	Close() error
}

// Request represents a single run-once client request.
type Request struct {
	OutputToStdout bool
}

// Client attempts to delegate run-once invocation to a resident server.
type Client interface {
	// TryRunOnce scans TCP range [49500,49550], performs handshake, and delegates to resident.
	// If no resident is found, returns delegated=false, err=nil.
	TryRunOnce(ctx context.Context, outputToStdout bool) (delegated bool, text string, err error)
}

// NewServer returns TCP implementation.
func NewServer() Server { return newTcpServer() }

// NewClient returns TCP implementation.
func NewClient() Client { return newTcpClient() }
