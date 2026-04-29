package planaction

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(plan *config.Plan) *cobra.Command {
	cfg := &config.PlanAction{Plan: plan}

	cmd := &cobra.Command{
		Use:     "action",
		Short:   "View plan actions (reusable building blocks for plan steps)",
		Aliases: []string{"a"},
	}

	cmd.AddCommand(
		newList(cfg),
		newGet(cfg),
	)

	return cmd
}
