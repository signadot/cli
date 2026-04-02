package plan

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/command/plantag"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	sdkplans "github.com/signadot/go-sdk/client/plans"
	"github.com/spf13/cobra"
)

func newRecompile(plan *config.Plan) *cobra.Command {
	cfg := &config.PlanRecompile{Plan: plan}

	cmd := &cobra.Command{
		Use:   "recompile PLAN_ID",
		Short: "Recompile a plan from its original prompt",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return recompile(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args[0])
		},
	}

	cfg.AddFlags(cmd)
	return cmd
}

func recompile(cfg *config.PlanRecompile, out, log io.Writer, planID string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	params := sdkplans.NewRecompilePlanParams().
		WithOrgName(cfg.Org).
		WithPlanID(planID)
	resp, err := cfg.Client.Plans.RecompilePlan(params, nil)
	if err != nil {
		return err
	}

	if cfg.Tag != "" {
		if _, err := plantag.ApplyTag(cfg.Plan, resp.Payload.ID, cfg.Tag); err != nil {
			return fmt.Errorf("plan recompiled (id=%s) but tagging failed: %w", resp.Payload.ID, err)
		}
		fmt.Fprintf(log, "Tagged plan %s as %q\n", resp.Payload.ID, cfg.Tag)
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
