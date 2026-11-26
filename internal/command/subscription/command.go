package subscription

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.Subscription{API: api}

	cmd := &cobra.Command{
		Use:     "subscription",
		Short:   "Inspect the org's subscription: plan, trial, and usage",
		Aliases: []string{"sub"},
	}

	// Subcommands
	cmd.AddCommand(
		newStatus(cfg),
		newUsage(cfg),
	)

	return cmd
}
