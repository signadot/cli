package jobs

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.Job{API: api}

	cmd := &cobra.Command{
		Use:   "job",
		Short: "Inspect and manipulate jobs",
	}

	// Subcommands
	cmd.AddCommand(
		newGet(cfg),
		newList(cfg),
		newSubmit(cfg),
		//newDelete(cfg),
	)

	return cmd
}
