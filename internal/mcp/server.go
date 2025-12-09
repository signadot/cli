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
func NewServer(cfg *config.MCP) *Server {
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

	// Create Server struct first (partial initialization)
	srv := &Server{
		log: log,
	}

	// Create remote manager (needed for the handler)
	srv.remoteManager = remote.NewRemoteManager(log, cfg.API)

	// Create MCP server with InitializedHandler to initilize the remote
	// manager. Note that in a stdio server, there's only one client session.
	serverOptions := &mcp.ServerOptions{
		InitializedHandler: func(ctx context.Context, req *mcp.InitializedRequest) {
			err := srv.remoteManager.Init(req.Session)
			if err != nil {
				log.Error("failed to initialize remote manager", "error", err)
			}
		},
		HasTools: true,
	}

	srv.mcpServer = mcp.NewServer(&mcp.Implementation{
		Name: "signadot-mcp-server",
	}, serverOptions)

	// Create authentication monitor
	srv.authMonitor = authmonitor.NewMonitor(cfg.API)

	// Create and setup tools
	srv.tools = tools.NewTools(log, srv.mcpServer, srv.remoteManager)
	return srv
}

func (s *Server) Run(ctx context.Context) error {
	// Setup tools
	s.tools.Setup()

	// Start remote metadata monitoring in the background
	initCh := make(chan struct{})
	s.remoteManager.SetCallback(func(ctx context.Context, meta *remote.Meta) {
		s.OnMetaChange(ctx, meta)
		select {
		case <-initCh:
		default:
			close(initCh)
		}
	})
	go s.remoteManager.Run(ctx, 30*time.Second)

	// Wait until the remote metadata is initialized to run the mcp server
	// (avoid sending tools/list_changed notifications before the remote
	// metadata is available)
	select {
	case <-initCh:
	case <-ctx.Done():
		return ctx.Err()
	}

	// Start authentication monitoring in the background
	s.authMonitor.SetCallback(s.OnAuthChange)
	go s.authMonitor.Run(ctx, 5*time.Second)

	// Run the mcp server
	return s.mcpServer.Run(ctx, &mcp.StdioTransport{})
}

func (s *Server) OnMetaChange(ctx context.Context, meta *remote.Meta) {
	// Update tools
	s.tools.Update(ctx, meta)
}

func (s *Server) OnAuthChange(ctx context.Context, _ bool) {
	// Update tools
	s.tools.Update(ctx, s.remoteManager.Meta())
}
