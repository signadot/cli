package plantag

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(plan *config.Plan) *cobra.Command {
	cfg := &config.PlanTag{Plan: plan}

	cmd := &cobra.Command{
		Use:     "tag",
		Short:   "Manage plan tags (named references to plans, like Docker tags)",
		Aliases: []string{"t"},
	}

	cmd.AddCommand(
		newList(cfg),
		newGet(cfg),
		newApply(cfg),
		newDelete(cfg),
	)

	return cmd
}
