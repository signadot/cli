package command

import (
	"github.com/signadot/cli/internal/command/cluster"
	"github.com/signadot/cli/internal/command/sandbox"
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cfg := &config.Api{}
	cobra.OnInitialize(cfg.Init)

	cmd := &cobra.Command{
		Use:   "signadot",
		Short: "Command-line interface for Signadot",

		// Don't print usage info automatically when errors occur.
		// Most of the time, the errors are not related to usage.
		SilenceUsage: true,
	}
	cfg.AddFlags(cmd)

	// Subcommands
	cmd.AddCommand(
		cluster.New(cfg),
		sandbox.New(cfg),
	)

	return cmd
}
