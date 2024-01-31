package resourceplugin

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.ResourcePlugin{API: api}

	cmd := &cobra.Command{
		Use:     "resourceplugin",
		Short:   "Inspect and manipulate resource plugins",
		Aliases: []string{"rp"},
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
