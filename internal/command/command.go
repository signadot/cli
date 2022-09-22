package command

import (
	"fmt"

	"github.com/signadot/cli/internal/buildinfo"
	"github.com/signadot/cli/internal/command/bug"
	"github.com/signadot/cli/internal/command/cluster"
	"github.com/signadot/cli/internal/command/sandbox"
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cfg := &config.API{}
	cobra.OnInitialize(cfg.Init)

	cmd := &cobra.Command{
		Use:     "signadot",
		Short:   "Command-line interface for Signadot",
		Version: fmt.Sprintf("%v (%v) - %v", buildinfo.Version, buildinfo.GitCommit, buildinfo.BuildDate),

		// Don't print usage info automatically when errors occur.
		// Most of the time, the errors are not related to usage.
		SilenceUsage: true,
	}
	cfg.AddFlags(cmd)

	// Subcommands
	cmd.AddCommand(
		cluster.New(cfg),
		sandbox.New(cfg),
		bug.New(cfg),
	)

	return cmd
}
