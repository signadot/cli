package local

import (
	"github.com/signadot/cli/internal/command/local/override"
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.Local{API: api}

	cmd := &cobra.Command{
		Use:   "local",
		Short: "Connect local machine with cluster",
	}

	// Subcommands
	cmd.AddCommand(
		newConnect(cfg),
		newStatus(cfg),
		newDisconnect(cfg),
		newProxy(cfg),
		override.New(cfg),
	)

	return cmd
}
