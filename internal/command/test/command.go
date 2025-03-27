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

	run := newRun(cfg)

	// Subcommands
	cmd.AddCommand(run)

	return cmd
}
