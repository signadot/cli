package remote

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/signadot/cli/internal/auth"
	"github.com/signadot/cli/internal/config"
)

// MetaOnChangeFunc is a callback function invoked when metadata changes.
type MetaOnChangeFunc func(ctx context.Context, meta *Meta) error

// Remote manages a single connection to a remote MCP server over HTTP.
// It maintains one client session and handles health checking and reconnection.
type Remote struct {
	mu sync.Mutex

	log      *slog.Logger
	cfg      *config.API
	client   *mcp.Client
	session  *mcp.ClientSession
	meta     *Meta // Cached metadata from the remote server
	onChange MetaOnChangeFunc
}

// NewRemoteManager creates a new Remote instance for managing connections
// to the remote MCP server.
func NewRemoteManager(log *slog.Logger, cfg *config.API) *Remote {
	return &Remote{
		log: log.With("component", "remote-manager"),
		cfg: cfg,
		client: mcp.NewClient(&mcp.Implementation{
			Name: "signadot-mcp-proxy",
		}, nil),
	}
}

func (r *Remote) SetCallback(onChange MetaOnChangeFunc) {
	r.onChange = onChange
}

// Meta returns the cached metadata. Returns nil if metadata hasn't been loaded
// yet.
func (r *Remote) Meta() *Meta {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.meta
}

// Session returns the existing session or creates a new one if needed.
// It performs a health check on existing sessions and automatically reconnects
// if the session is unhealthy. The session is authenticated using the current
// auth configuration.
func (r *Remote) Session(ctx context.Context) (*mcp.ClientSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if we have an existing session and verify it's healthy
	if r.session != nil {
		// Perform a lightweight health check using ping with a short timeout
		healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		err := r.session.Ping(healthCtx, nil)
		if err != nil {
			r.log.Debug("remote session is unhealthy, recreating", "error", err)
			// Session is unhealthy, close it and create a new one
			r.session.Close()
			r.session = nil
		} else {
			// Session is healthy, return it
			return r.session, nil
		}
	}

	// Resolve authentication information
	authInfo, err := auth.ResolveAuth()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve auth: %w", err)
	}
	if !auth.IsAuthenticated(authInfo) {
		return nil, fmt.Errorf("not authenticated")
	}

	// Create HTTP transport with authentication headers
	transport := &mcp.StreamableClientTransport{
		Endpoint: r.cfg.MCPURL + "/stream",
		HTTPClient: &http.Client{
			Transport: &authTransport{
				RoundTripper: http.DefaultTransport,
				authInfo:     authInfo,
			},
		},
	}

	// Connect to the remote MCP server
	sess, err := r.client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote server: %w", err)
	}

	// Store the session for future use
	r.log.Debug("remote session created", "sessionID", sess.ID())
	r.session = sess
	return sess, nil
}

// Close closes the current session and releases resources.
// It is safe to call Close even if no session exists.
func (r *Remote) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.session != nil {
		r.session.Close()
		r.session = nil
	}
}

// Run periodically fetches and updates metadata from the remote server.
// It runs until the context is cancelled.
func (r *Remote) Run(ctx context.Context, checkInterval time.Duration) error {
	r.log.Debug("remote metadata updater started")

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	// Fetch immediately on start
	if err := r.updateMeta(ctx); err != nil {
		r.log.Error("failed to fetch remote metadata", "error", err)
	}

	// Periodically fetch metadata
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := r.updateMeta(ctx); err != nil {
				r.log.Error("failed to fetch remote metadata", "error", err)
			}
		}
	}
}
