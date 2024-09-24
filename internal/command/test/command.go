package test

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.Test{API: api}
	cmd := &cobra.Command{
		Use:     "test",
		Short:   "Signadot tests",
		Aliases: []string{"t"},
		Hidden:  true,
	}

	apply := newApply(cfg)
	get := newGet(cfg)
	lst := newList(cfg)
	del := newDelete(cfg)

	// Subcommands
	cmd.AddCommand(get, lst, del, apply)

	return cmd
}
