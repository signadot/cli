package auth

import (
	"context"
	"sync"
	"time"
)

// Monitor tracks authentication status and invokes a callback when it changes.
type Monitor struct {
	onChange  func(authenticated bool)
	mu        sync.Mutex
	lastState bool
}

// NewMonitor creates a new authentication monitor with the given change
// callback.
func NewMonitor(onChange func(authenticated bool)) *Monitor {
	return &Monitor{
		onChange:  onChange,
		lastState: false,
	}
}

// Run begins monitoring authentication status. It performs an initial check and
// then checks periodically at the specified interval until the context is
// cancelled.
func (m *Monitor) Run(ctx context.Context, checkInterval time.Duration) {
	// Initial check
	m.Check()

	// Periodic checks
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.Check()
		}
	}
}

// Check verifies the current authentication status and invokes the onChange
// callback if the status has changed since the last check.
func (m *Monitor) Check() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check current auth status
	authInfo, err := ResolveAuth()
	currentState := err == nil && IsAuthenticated(authInfo)

	// Only update if state changed
	if currentState != m.lastState {
		m.onChange(currentState)
		m.lastState = currentState
	}
}
