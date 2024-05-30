package jobrunnergroup

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.JobRunnerGroup{API: api}

	cmd := &cobra.Command{
		Use:     "jobrunnergroup",
		Short:   "Inspect and manipulate jobrunnergroup",
		Aliases: []string{"jrg"},
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
