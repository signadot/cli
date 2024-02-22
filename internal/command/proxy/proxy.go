package proxy

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.Proxy{API: api}

	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "Forward proxy to cluster services",
	}

	// Subcommands
	cmd.AddCommand(
		newConnect(cfg),
	)

	return cmd
}
