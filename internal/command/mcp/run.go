package mcp

import (
	"context"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/mcp"
	"github.com/spf13/cobra"
)

func newRun(mcpConfig *config.MCP) *cobra.Command {
	cfg := &config.MCPRun{
		MCP: mcpConfig,
	}
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run MCP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), cfg)
		},
	}
	cfg.AddFlags(cmd)
	return cmd
}

func run(ctx context.Context, cfg *config.MCPRun) error {
	srv := mcp.NewServer(cfg)
	if err := srv.Run(ctx); err != nil {
		return err
	}
	return nil
}
