package monitor

import (
	"context"
	"sync"
	"time"

	"github.com/signadot/cli/internal/auth"
	"github.com/signadot/cli/internal/config"
)

type MonitorOnChangeFunc func(ctx context.Context, authenticated bool) error

// Monitor tracks authentication status and invokes a callback when it changes.
type Monitor struct {
	cfg       *config.API
	onChange  MonitorOnChangeFunc
	mu        sync.Mutex
	lastState bool
}

// NewMonitor creates a new authentication monitor with the given change
// callback.
func NewMonitor(cfg *config.API) *Monitor {
	return &Monitor{
		cfg:       cfg,
		lastState: false,
	}
}

func (m *Monitor) SetCallback(onChange MonitorOnChangeFunc) {
	m.onChange = onChange
}

// Run begins monitoring authentication status. It performs an initial check and
// then checks periodically at the specified interval until the context is
// cancelled.
func (m *Monitor) Run(ctx context.Context, checkInterval time.Duration) {
	// Initial check
	m.Check(ctx)

	// Periodic checks
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.Check(ctx)
		}
	}
}

// Check verifies the current authentication status and invokes the onChange
// callback if the status has changed since the last check.
func (m *Monitor) Check(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Make sure the auth token is refreshed
	m.cfg.RefreshAPIConfig()

	// Check current auth status
	authInfo, err := auth.ResolveAuth()
	currentState := err == nil && auth.IsAuthenticated(authInfo)

	// Only update if state changed
	if currentState != m.lastState {
		err := m.onChange(ctx, currentState)
		if err != nil {
			// don't update the state if the callback fails
			return
		}
		m.lastState = currentState
	}
}
