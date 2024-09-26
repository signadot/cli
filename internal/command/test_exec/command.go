package test_exec

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.TestExec{API: api}
	cmd := &cobra.Command{
		Use:     "testx",
		Short:   "Execute Signadot tests",
		Aliases: []string{"tx"},
		Hidden:  true,
	}

	// Subcommands
	cmd.AddCommand(
		//newRun(cfg),
		newCancel(cfg),
		newGet(cfg),
		newList(cfg),
	)

	return cmd
}
