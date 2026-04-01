package plan

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	sdkplans "github.com/signadot/go-sdk/client/plans"
	"github.com/spf13/cobra"
)

func newGet(plan *config.Plan) *cobra.Command {
	cfg := &config.PlanGet{Plan: plan}

	cmd := &cobra.Command{
		Use:   "get PLAN_ID",
		Short: "Get plan details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return getPlan(cfg, cmd.OutOrStdout(), args[0])
		},
	}

	return cmd
}

func getPlan(cfg *config.PlanGet, out io.Writer, planID string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	params := sdkplans.NewGetPlanParams().
		WithOrgName(cfg.Org).
		WithPlanID(planID)
	resp, err := cfg.Client.Plans.GetPlan(params, nil)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printPlanDetails(out, resp.Payload)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
