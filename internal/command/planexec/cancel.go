package planexec

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	planexecs "github.com/signadot/go-sdk/client/plan_executions"
	"github.com/spf13/cobra"
)

func newCancel(exec *config.PlanExecution) *cobra.Command {
	cfg := &config.PlanExecCancel{PlanExecution: exec}

	cmd := &cobra.Command{
		Use:   "cancel EXECUTION_ID",
		Short: "Cancel a running plan execution",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cancelExec(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args[0])
		},
	}

	return cmd
}

func cancelExec(cfg *config.PlanExecCancel, out, log io.Writer, execID string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	params := planexecs.NewCancelPlanExecutionParams().
		WithOrgName(cfg.Org).
		WithPlanExecutionID(execID)
	resp, err := cfg.Client.PlanExecutions.CancelPlanExecution(params, nil)
	if err != nil {
		return err
	}
	fmt.Fprintf(log, "Cancelled execution %q.\n", execID)

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printExecDetails(out, resp.Payload)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
