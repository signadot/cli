package auth

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.Auth{API: api}

	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication commands",
	}

	// Subcommands
	cmd.AddCommand(
		newLogin(cfg),
		newStatus(cfg),
		newLogout(cfg),
	)

	return cmd
}
