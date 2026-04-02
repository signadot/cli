package planexec

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	planexecs "github.com/signadot/go-sdk/client/plan_executions"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newOutputs(exec *config.PlanExecution) *cobra.Command {
	cfg := &config.PlanExecOutputs{PlanExecution: exec}

	cmd := &cobra.Command{
		Use:   "outputs EXECUTION_ID",
		Short: "List outputs of a plan execution",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return listOutputs(cfg, cmd.OutOrStdout(), args[0])
		},
	}

	return cmd
}

func listOutputs(cfg *config.PlanExecOutputs, out io.Writer, execID string) error {
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

	var outputs []*models.PlanOutputStatus
	if resp.Payload.Status != nil {
		outputs = resp.Payload.Status.Outputs
	}
	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printOutputsTable(out, outputs)
	case config.OutputFormatJSON:
		return print.RawJSON(out, outputs)
	case config.OutputFormatYAML:
		return print.RawYAML(out, outputs)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
