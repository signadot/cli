package mcp

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.MCP{API: api}
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "MCP server commands",
	}

	// Subcommands
	cmd.AddCommand(
		newRun(cfg),
	)

	return cmd
}
