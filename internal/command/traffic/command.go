package traffic

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.Traffic{API: api}

	cmd := &cobra.Command{
		Use:     "traffic",
		Short:   "Operations on sandbox traffic",
		Aliases: []string{"tr"},
	}

	// Subcommands
	cmd.AddCommand(
		newRecord(cfg),
	)

	return cmd
}
