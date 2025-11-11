package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/mcp/tools"
)

type Server struct {
	*mcp.Server
}

func NewServer(cfg *config.MCPRun) *Server {
	srv := &Server{
		Server: mcp.NewServer(&mcp.Implementation{Name: "greeter"}, nil),
	}
	tools.Setup(cfg, srv.Server)
	return srv
}

func (s *Server) Run(ctx context.Context) error {
	return s.Server.Run(ctx, &mcp.StdioTransport{})
}
