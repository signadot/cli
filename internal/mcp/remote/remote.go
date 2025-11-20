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
		}, &mcp.ClientOptions{
			KeepAlive: 10 * time.Second,
		}),
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
// When KeepAlive is enabled, it automatically handles health checks and closes
// the session if pings fail. This method will recreate the session if it was
// closed by KeepAlive or if no session exists.
func (r *Remote) Session() (*mcp.ClientSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// If we have a session, return it. KeepAlive handles health checks automatically
	// and will close the session if pings fail. If the session was closed by KeepAlive,
	// operations will fail with ErrConnectionClosed and the tool handler will recreate it.
	if r.session != nil {
		return r.session, nil
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

	// Connect to the remote MCP server (use background context to avoid context
	// cancellation errors, this session will be used across multiple tool
	// calls)
	sess, err := r.client.Connect(context.Background(), transport, nil)
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
