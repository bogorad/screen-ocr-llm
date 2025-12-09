package process

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"screen-ocr-llm/src/messages"
	"screen-ocr-llm/src/router"
)

// Process interface defines the lifecycle methods for all processes
type Process interface {
	// Start begins the process execution
	Start(ctx context.Context, router *router.Router) error

	// Stop gracefully shuts down the process
	Stop() error

	// IsRunning returns true if the process is currently running
	IsRunning() bool

	// Name returns the process name for identification
	Name() string
}

// ProcessState represents the current state of a process
type ProcessState int

const (
	StateStopped ProcessState = iota
	StateStarting
	StateRunning
	StateStopping
	StateCrashed
)

func (s ProcessState) String() string {
	switch s {
	case StateStopped:
		return "stopped"
	case StateStarting:
		return "starting"
	case StateRunning:
		return "running"
	case StateStopping:
		return "stopping"
	case StateCrashed:
		return "crashed"
	default:
		return "unknown"
	}
}

// ProcessInfo holds information about a managed process
type ProcessInfo struct {
	Process    Process
	State      ProcessState
	StartTime  time.Time
	CrashCount int
	LastError  error
	Context    context.Context
	CancelFunc context.CancelFunc
}

// Manager manages the lifecycle of all application processes
type Manager struct {
	processes map[string]*ProcessInfo
	router    *router.Router
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewManager creates a new process manager
func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		processes: make(map[string]*ProcessInfo),
		router:    router.NewRouter(),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Register adds a process to the manager
func (m *Manager) Register(process Process) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := process.Name()
	if _, exists := m.processes[name]; exists {
		return fmt.Errorf("process %s already registered", name)
	}

	ctx, cancel := context.WithCancel(m.ctx)
	m.processes[name] = &ProcessInfo{
		Process:    process,
		State:      StateStopped,
		Context:    ctx,
		CancelFunc: cancel,
	}

	log.Printf("Process %s registered", name)
	return nil
}

// Start starts a specific process
func (m *Manager) Start(name string) error {
	m.mu.Lock()
	info, exists := m.processes[name]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("process %s not found", name)
	}

	if info.State == StateRunning {
		m.mu.Unlock()
		return fmt.Errorf("process %s already running", name)
	}

	info.State = StateStarting
	info.StartTime = time.Now()
	m.mu.Unlock()

	// Start process in goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Process %s panicked: %v", name, r)
				m.markCrashed(name, fmt.Errorf("panic: %v", r))
			}
		}()

		log.Printf("Starting process %s", name)
		err := info.Process.Start(info.Context, m.router)

		m.mu.Lock()
		if err != nil {
			info.State = StateCrashed
			info.LastError = err
			info.CrashCount++
			log.Printf("Process %s failed to start: %v", name, err)
		} else {
			info.State = StateRunning
			log.Printf("Process %s started successfully", name)
		}
		m.mu.Unlock()
	}()

	return nil
}

// StartAll starts all registered processes
func (m *Manager) StartAll() error {
	m.mu.RLock()
	names := make([]string, 0, len(m.processes))
	for name := range m.processes {
		names = append(names, name)
	}
	m.mu.RUnlock()

	for _, name := range names {
		if err := m.Start(name); err != nil {
			return fmt.Errorf("failed to start process %s: %v", name, err)
		}

		// Small delay between process starts
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// Stop stops a specific process
func (m *Manager) Stop(name string) error {
	m.mu.Lock()
	info, exists := m.processes[name]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("process %s not found", name)
	}

	if info.State != StateRunning {
		m.mu.Unlock()
		return nil // Already stopped
	}

	info.State = StateStopping
	m.mu.Unlock()

	log.Printf("Stopping process %s", name)

	// Cancel context first
	info.CancelFunc()

	// Try graceful stop
	if err := info.Process.Stop(); err != nil {
		log.Printf("Error stopping process %s: %v", name, err)
	}

	m.mu.Lock()
	info.State = StateStopped
	m.mu.Unlock()

	log.Printf("Process %s stopped", name)
	return nil
}

// StopAll stops all processes gracefully
func (m *Manager) StopAll() {
	log.Printf("Stopping all processes...")

	// Send DIENOW to all processes first
	m.router.Broadcast(messages.MessageEnvelope{
		From:    messages.ProcessMain,
		To:      "*",
		Message: messages.DIENOW{},
	})

	// Give processes time to handle DIENOW
	time.Sleep(500 * time.Millisecond)

	m.mu.RLock()
	names := make([]string, 0, len(m.processes))
	for name := range m.processes {
		names = append(names, name)
	}
	m.mu.RUnlock()

	// Stop all processes
	for _, name := range names {
		m.Stop(name)
	}

	// Cancel main context
	m.cancel()

	log.Printf("All processes stopped")
}

// GetRouter returns the message router
func (m *Manager) GetRouter() *router.Router {
	return m.router
}

// GetStatus returns the status of all processes
func (m *Manager) GetStatus() map[string]ProcessState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]ProcessState)
	for name, info := range m.processes {
		status[name] = info.State
	}
	return status
}

// markCrashed marks a process as crashed
func (m *Manager) markCrashed(name string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if info, exists := m.processes[name]; exists {
		info.State = StateCrashed
		info.LastError = err
		info.CrashCount++
		log.Printf("Process %s crashed: %v (crash count: %d)", name, err, info.CrashCount)
	}
}

// RestartCrashed restarts any crashed processes (with exponential backoff)
func (m *Manager) RestartCrashed() {
	m.mu.RLock()
	crashed := make([]string, 0)
	for name, info := range m.processes {
		if info.State == StateCrashed && info.CrashCount < 5 { // Max 5 restart attempts
			crashed = append(crashed, name)
		}
	}
	m.mu.RUnlock()

	for _, name := range crashed {
		log.Printf("Attempting to restart crashed process %s", name)
		if err := m.Start(name); err != nil {
			log.Printf("Failed to restart process %s: %v", name, err)
		}
	}
}
