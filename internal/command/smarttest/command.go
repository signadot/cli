package smarttest

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.SmartTest{API: api}
	cmd := &cobra.Command{
		Use:     "smart-test",
		Short:   "Signadot smart tests",
		Aliases: []string{"st"},
	}

	run := newRun(cfg)
	exec := newExecution(cfg)

	// Subcommands
	cmd.AddCommand(run)
	cmd.AddCommand(exec)
	return cmd
}
