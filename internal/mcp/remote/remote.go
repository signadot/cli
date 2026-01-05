package remote

import (
	"context"
	"errors"
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
type MetaOnChangeFunc func(ctx context.Context, meta *Meta)

// Remote manages a single connection to a remote MCP server over HTTP.
// It maintains one client session and handles health checking and reconnection.
type Remote struct {
	mu sync.Mutex

	log           *slog.Logger
	mcpCfg        *config.MCP
	remoteClient  *mcp.Client
	remoteSession *mcp.ClientSession
	localSession  *mcp.ServerSession
	meta          *Meta // Cached metadata from the remote server
	onChange      MetaOnChangeFunc
}

// NewRemoteManager creates a new Remote instance for managing connections
// to the remote MCP server. The client is created lazily when capabilities are known.
func NewRemoteManager(log *slog.Logger, mcpCfg *config.MCP) *Remote {
	return &Remote{
		log:    log.With("component", "remote-manager"),
		mcpCfg: mcpCfg,
	}
}

func (r *Remote) SetCallback(onChange MetaOnChangeFunc) {
	r.onChange = onChange
}

func (r *Remote) Init(localSession *mcp.ServerSession) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.remoteClient != nil {
		return errors.New("remote client already initialized")
	}

	// get the initialize params from the local session
	initParams := localSession.InitializeParams()

	// lets create the remote client

	// define the implementation
	var impl *mcp.Implementation
	if initParams.ClientInfo != nil {
		impl = initParams.ClientInfo
	} else {
		// provide a default implementation if not provided
		impl = &mcp.Implementation{
			Name: "signadot-mcp-proxy",
		}
	}

	// create the client options
	opts := &mcp.ClientOptions{
		KeepAlive: 10 * time.Second,
	}
	if !r.mcpCfg.DisableElicitation {
		if initParams.Capabilities != nil {
			clientCaps := initParams.Capabilities
			// If the local client supports elicitation and elicitation is not
			// disabled, set up a handler to proxy elicitation requests
			if clientCaps.Elicitation != nil {
				opts.ElicitationHandler = r.proxyElicitation
				r.log.Debug("elicitation handler configured for remote client")
			}
		}
	} else {
		r.log.Debug("elicitation disabled via --disable-elicitation flag")
	}

	r.localSession = localSession
	r.remoteClient = mcp.NewClient(impl, opts)
	return nil
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
	if r.remoteSession != nil {
		return r.remoteSession, nil
	}

	// Ensure client is initialized
	if r.remoteClient == nil {
		return nil, fmt.Errorf("client hasn't been initialized, cannot create remote session")
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
		Endpoint: r.mcpCfg.API.MCPURL + "/stream",
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
	sess, err := r.remoteClient.Connect(context.Background(), transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote server: %w", err)
	}

	// Store the session for future use
	r.log.Debug("remote session created", "sessionID", sess.ID())
	r.remoteSession = sess
	return sess, nil
}

// Close closes the current session and releases resources.
// It is safe to call Close even if no session exists.
func (r *Remote) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.remoteSession != nil {
		r.remoteSession.Close()
		r.remoteSession = nil
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

func (r *Remote) proxyElicitation(ctx context.Context, req *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
	// Proxy the elicitation request from the remote server to the local client
	r.log.Debug("proxying elicitation request to local client",
		"message", req.Params.Message)

	// Forward the elicitation request to the local client
	result, err := r.localSession.Elicit(ctx, req.Params)
	if err != nil {
		r.log.Error("failed to proxy elicitation to local client", "error", err)
		return nil, err
	}

	r.log.Debug("elicitation response received from local client",
		"action", result.Action)
	return result, nil
}
