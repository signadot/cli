package mcp

import (
	"context"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/mcp"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.MCP{
		API: api,
	}
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run MCP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), cfg)
		},
	}
	cfg.AddFlags(cmd)
	return cmd
}

func run(ctx context.Context, cfg *config.MCP) error {
	// initialize api config
	if err := cfg.API.InitUnauthAPIConfig(); err != nil {
		return err
	}

	// create mcp server
	srv := mcp.NewServer(cfg)

	// run mcp server
	return srv.Run(ctx)
}
