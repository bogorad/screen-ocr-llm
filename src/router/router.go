package router

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"screen-ocr-llm/src/messages"
)

// ChannelInfo holds information about a process channel
type ChannelInfo struct {
	Channel   chan messages.MessageEnvelope
	ProcessID string
	Active    bool
}

// Router handles message routing between processes
type Router struct {
	channels    map[string]*ChannelInfo
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	logMessages bool
}

// NewRouter creates a new message router
func NewRouter() *Router {
	ctx, cancel := context.WithCancel(context.Background())
	return &Router{
		channels:    make(map[string]*ChannelInfo),
		ctx:         ctx,
		cancel:      cancel,
		logMessages: true, // Enable message logging for debugging
	}
}

// RegisterProcess registers a process with the router
func (r *Router) RegisterProcess(processID string, bufferSize int) (<-chan messages.MessageEnvelope, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.channels[processID]; exists {
		return nil, fmt.Errorf("process %s already registered", processID)
	}

	ch := make(chan messages.MessageEnvelope, bufferSize)
	r.channels[processID] = &ChannelInfo{
		Channel:   ch,
		ProcessID: processID,
		Active:    true,
	}

	log.Printf("Router: Registered process %s with buffer size %d", processID, bufferSize)
	return ch, nil
}

// UnregisterProcess removes a process from the router
func (r *Router) UnregisterProcess(processID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if info, exists := r.channels[processID]; exists {
		info.Active = false
		close(info.Channel)
		delete(r.channels, processID)
		log.Printf("Router: Unregistered process %s", processID)
	}
}

// Send sends a message to a specific process
func (r *Router) Send(envelope messages.MessageEnvelope) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.logMessages {
		log.Printf("Router: %s -> %s: %s", envelope.From, envelope.To, envelope.Message.Type())
	}

	// Handle broadcast messages
	if envelope.To == "*" {
		return r.broadcastMessage(envelope)
	}

	// Send to specific process
	info, exists := r.channels[envelope.To]
	if !exists {
		return fmt.Errorf("process %s not found", envelope.To)
	}

	if !info.Active {
		return fmt.Errorf("process %s is not active", envelope.To)
	}

	// Non-blocking send with timeout
	select {
	case info.Channel <- envelope:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout sending message to process %s", envelope.To)
	case <-r.ctx.Done():
		return fmt.Errorf("router is shutting down")
	}
}

// Broadcast sends a message to all registered processes
func (r *Router) Broadcast(envelope messages.MessageEnvelope) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.logMessages {
		log.Printf("Router: Broadcasting %s from %s", envelope.Message.Type(), envelope.From)
	}

	r.broadcastMessage(envelope)
}

// broadcastMessage sends a message to all active processes (internal helper)
func (r *Router) broadcastMessage(envelope messages.MessageEnvelope) error {
	var errors []string

	for processID, info := range r.channels {
		if !info.Active || processID == envelope.From {
			continue // Skip inactive processes and sender
		}

		// Create a copy of the envelope for each recipient
		envCopy := messages.MessageEnvelope{
			From:    envelope.From,
			To:      processID,
			Message: envelope.Message,
		}

		// Non-blocking send
		select {
		case info.Channel <- envCopy:
			// Success
		case <-time.After(1 * time.Second): // Shorter timeout for broadcast
			errors = append(errors, fmt.Sprintf("timeout sending to %s", processID))
		case <-r.ctx.Done():
			return fmt.Errorf("router is shutting down")
		}
	}

	if len(errors) > 0 {
		log.Printf("Router: Broadcast errors: %v", errors)
	}

	return nil
}

// SendToMain is a convenience method for sending messages to the main process
func (r *Router) SendToMain(from string, message messages.Message) error {
	return r.Send(messages.MessageEnvelope{
		From:    from,
		To:      messages.ProcessMain,
		Message: message,
	})
}

// GetActiveProcesses returns a list of active process IDs
func (r *Router) GetActiveProcesses() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var active []string
	for processID, info := range r.channels {
		if info.Active {
			active = append(active, processID)
		}
	}
	return active
}

// GetChannelStats returns statistics about message channels
func (r *Router) GetChannelStats() map[string]int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := make(map[string]int)
	for processID, info := range r.channels {
		if info.Active {
			stats[processID] = len(info.Channel)
		}
	}
	return stats
}

// SetMessageLogging enables or disables message logging
func (r *Router) SetMessageLogging(enabled bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logMessages = enabled
}

// Shutdown gracefully shuts down the router
func (r *Router) Shutdown() {
	log.Printf("Router: Shutting down...")

	r.cancel()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Close all channels
	for processID, info := range r.channels {
		if info.Active {
			info.Active = false
			close(info.Channel)
			log.Printf("Router: Closed channel for process %s", processID)
		}
	}

	// Clear channels map
	r.channels = make(map[string]*ChannelInfo)

	log.Printf("Router: Shutdown complete")
}

// IsHealthy returns true if the router is functioning properly
func (r *Router) IsHealthy() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	select {
	case <-r.ctx.Done():
		return false
	default:
		return true
	}
}

// WaitForMessage waits for a specific message type from a channel with timeout
func WaitForMessage(ch <-chan messages.MessageEnvelope, messageType string, timeout time.Duration) (messages.MessageEnvelope, error) {
	deadline := time.After(timeout)

	for {
		select {
		case envelope := <-ch:
			if envelope.Message.Type() == messageType {
				return envelope, nil
			}
			// Continue waiting for the right message type
		case <-deadline:
			return messages.MessageEnvelope{}, fmt.Errorf("timeout waiting for message type %s", messageType)
		}
	}
}

// DrainChannel drains all messages from a channel (useful for cleanup)
func DrainChannel(ch <-chan messages.MessageEnvelope) int {
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			return count
		}
	}
}
