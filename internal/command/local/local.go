package local

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.Local{API: api}

	cmd := &cobra.Command{
		Use:   "local",
		Short: "connect with sandboxes locally",
	}

	// Subcommands
	cmd.AddCommand(
		newConnect(cfg),
		newDisconnect(cfg),
	)

	return cmd
}