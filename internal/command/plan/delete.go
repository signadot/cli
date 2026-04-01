package plan

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	sdkplans "github.com/signadot/go-sdk/client/plans"
	"github.com/spf13/cobra"
)

func newDelete(plan *config.Plan) *cobra.Command {
	cfg := &config.PlanDelete{Plan: plan}

	cmd := &cobra.Command{
		Use:   "delete PLAN_ID",
		Short: "Delete a plan",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deletePlan(cfg, cmd.ErrOrStderr(), args[0])
		},
	}

	return cmd
}

func deletePlan(cfg *config.PlanDelete, log io.Writer, planID string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	params := sdkplans.NewDeletePlanParams().
		WithOrgName(cfg.Org).
		WithPlanID(planID)
	_, err := cfg.Client.Plans.DeletePlan(params, nil)
	if err != nil {
		return err
	}
	fmt.Fprintf(log, "Deleted plan %q.\n", planID)
	return nil
}
