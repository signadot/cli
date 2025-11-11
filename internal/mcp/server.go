package mcp

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/signadot/cli/internal/auth"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/mcp/tools"
)

type Server struct {
	*mcp.Server
	authMonitor *auth.Monitor
}

// NewServer creates a new MCP server and registers tools.
// Authentication is checked at startup, monitored continuously, and checked
// whenever list_tools is called (for each session). Authenticated tools are
// dynamically added/removed based on authentication status, and clients are
// automatically notified via tools/list_changed notifications.
func NewServer(cfg *config.MCPRun) *Server {
	srv := &Server{
		Server: mcp.NewServer(&mcp.Implementation{Name: "greeter"}, nil),
	}

	// Create authentication monitor
	srv.authMonitor = auth.NewMonitor(srv.OnAuthChange)

	// Add middleware to check authentication when list_tools is called
	// This ensures tools are up-to-date for each session
	srv.Server.AddReceivingMiddleware(CreateAuthCheckMiddleware(srv.authMonitor))

	// Setup tools
	tools.Setup(cfg, srv.Server)

	return srv
}

func (s *Server) Run(ctx context.Context) error {
	// Check authentication before the server starts
	s.authMonitor.Check()

	// Start authentication monitoring in the background
	go s.authMonitor.Run(ctx, 5*time.Second)

	// Run the mcp server
	return s.Server.Run(ctx, &mcp.StdioTransport{})
}

func (s *Server) OnAuthChange(isAuth bool) {
	// Update tools
	tools.Update(s.Server)
}
