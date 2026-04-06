package planexec

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(plan *config.Plan) *cobra.Command {
	cfg := &config.PlanExecution{Plan: plan}

	cmd := &cobra.Command{
		Use:     "execution",
		Short:   "Manage plan executions (runs of compiled plans)",
		Aliases: []string{"x"},
	}

	cmd.AddCommand(
		newList(cfg),
		newGet(cfg),
		newCancel(cfg),
		newOutputs(cfg),
		newGetOutput(cfg),
		newLogs(cfg),
	)

	return cmd
}
