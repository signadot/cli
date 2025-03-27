package synthetic

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.Synthetic{API: api}
	cmd := &cobra.Command{
		Use:     "synthetic",
		Short:   "Signadot synthetic tests",
		Aliases: []string{"t"},
		Hidden:  true,
	}

	apply := newApply(cfg)
	get := newGet(cfg)
	lst := newList(cfg)
	del := newDelete(cfg)
	run := newRun(cfg)

	// Subcommands
	cmd.AddCommand(run, get, lst, del, apply)

	return cmd
}
