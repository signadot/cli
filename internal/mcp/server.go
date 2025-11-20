package mcp

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	authmonitor "github.com/signadot/cli/internal/auth/monitor"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/mcp/remote"
	"github.com/signadot/cli/internal/mcp/tools"
)

type Server struct {
	log           *slog.Logger
	mcpServer     *mcp.Server
	authMonitor   *authmonitor.Monitor
	remoteManager *remote.Remote
	tools         *tools.Tools
}

// NewServer creates a new MCP server and registers tools.
// Authentication is checked at startup and monitored continuously in the background.
// Authenticated tools are dynamically added/removed based on authentication status,
// and clients are automatically notified via tools/list_changed notifications.
func NewServer(cfg *config.MCPRun) *Server {
	// write logs to sterr (for an mcp server in stdio mode).
	logLevel := slog.LevelInfo
	if cfg.Debug {
		logLevel = slog.LevelDebug
	}
	log := slog.New(
		slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: logLevel,
		}),
	)

	srv := &Server{
		log: log,
		// Create MCP server
		mcpServer: mcp.NewServer(&mcp.Implementation{Name: "signadot-mcp-server"}, nil),
	}

	// Create authentication monitor
	srv.authMonitor = authmonitor.NewMonitor(cfg.API)

	// Create remote manager
	srv.remoteManager = remote.NewRemoteManager(log, cfg.API)

	// Create and setup tools
	srv.tools = tools.NewTools(log, srv.mcpServer, srv.remoteManager)
	return srv
}

func (s *Server) Run(ctx context.Context) error {
	// Setup tools
	s.tools.Setup()

	// Start authentication monitoring in the background
	s.authMonitor.SetCallback(s.OnAuthChange)
	go s.authMonitor.Run(ctx, 5*time.Second)

	// Start remote metadata monitoring in the background
	initCh := make(chan struct{})
	s.remoteManager.SetCallback(func(ctx context.Context, meta *remote.Meta) error {
		err := s.OnMetaChange(ctx, meta)
		select {
		case <-initCh:
		default:
			close(initCh)
		}
		return err
	})
	go s.remoteManager.Run(ctx, 30*time.Second)

	// Give some time for the remote metadata to be initialized
	select {
	case <-initCh:
	case <-time.After(500 * time.Millisecond):
	case <-ctx.Done():
		return ctx.Err()
	}

	// Run the mcp server
	return s.mcpServer.Run(ctx, &mcp.StdioTransport{})
}

func (s *Server) OnMetaChange(ctx context.Context, meta *remote.Meta) error {
	// Update tools
	return s.tools.Update(ctx, meta)
}

func (s *Server) OnAuthChange(ctx context.Context, _ bool) error {
	// Update tools
	return s.tools.Update(ctx, s.remoteManager.Meta())
}
