package sandbox

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.Sandbox{API: api}

	cmd := &cobra.Command{
		Use:     "sandbox",
		Short:   "Inspect and manipulate sandboxes",
		Aliases: []string{"sb"},
	}

	// Subcommands
	cmd.AddCommand(
		newGet(cfg),
		newList(cfg),
		newApply(cfg),
		newDelete(cfg),
	)

	return cmd
}
