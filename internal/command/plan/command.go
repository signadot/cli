package plan

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.Plan{API: api}

	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Manage plans",
	}

	// Subcommands
	cmd.AddCommand(
		newCompile(cfg),
		newCreate(cfg),
		newList(cfg),
		newGet(cfg),
		newDelete(cfg),
	)

	return cmd
}
