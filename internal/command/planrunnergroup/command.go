package planrunnergroup

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.PlanRunnerGroup{API: api}

	cmd := &cobra.Command{
		Use:     "planrunnergroup",
		Short:   "Manage plan runner groups",
		Aliases: []string{"prg"},
		Hidden:  true,
	}

	// Subcommands
	cmd.AddCommand(
		newGet(cfg),
		newList(cfg),
		newApply(cfg),
		newDelete(cfg),
		newImage(cfg),
	)

	return cmd
}
