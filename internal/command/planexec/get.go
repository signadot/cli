package planexec

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	planexecs "github.com/signadot/go-sdk/client/plan_executions"
	sdkplans "github.com/signadot/go-sdk/client/plans"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newGet(exec *config.PlanExecution) *cobra.Command {
	cfg := &config.PlanExecGet{PlanExecution: exec}

	cmd := &cobra.Command{
		Use:   "get EXECUTION_ID",
		Short: "Get plan execution details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return getExec(cfg, cmd.OutOrStdout(), args[0])
		},
	}

	return cmd
}

func getExec(cfg *config.PlanExecGet, out io.Writer, execID string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	params := planexecs.NewGetPlanExecutionParams().
		WithOrgName(cfg.Org).
		WithExecutionID(execID)
	resp, err := cfg.Client.PlanExecutions.GetPlanExecution(params, nil)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printExecDetails(out, resp.Payload, fetchPlanSpec(cfg.API, resp.Payload))
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}

// fetchPlanSpec returns the spec of the plan referenced by ex, or nil
// if the plan can't be fetched (deleted, network error, etc). The
// detail printer renders without resolved values when the spec is nil.
func fetchPlanSpec(api *config.API, ex *models.PlanExecution) *models.PlanSpec {
	if ex == nil || ex.Spec == nil || ex.Spec.PlanID == "" {
		return nil
	}
	params := sdkplans.NewGetPlanParams().
		WithOrgName(api.Org).
		WithPlanID(ex.Spec.PlanID)
	resp, err := api.Client.Plans.GetPlan(params, nil)
	if err != nil || resp.Payload == nil {
		return nil
	}
	return resp.Payload.Spec
}
