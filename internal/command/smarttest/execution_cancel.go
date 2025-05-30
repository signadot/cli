package smarttest

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/client/test_executions"
	"github.com/spf13/cobra"
)

func newCancel(tConfig *config.SmartTestExec) *cobra.Command {
	cfg := &config.SmartTestExecCancel{
		SmartTestExec: tConfig,
	}
	cmd := &cobra.Command{
		Use:   "cancel [ID | --run-id RUN_ID]",
		Short: "Cancel a test execution",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cancel(cmd.Context(), cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}
	cfg.AddFlags(cmd)
	return cmd
}

func cancel(ctx context.Context, cfg *config.SmartTestExecCancel,
	wOut, wErr io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	if err := validateCancel(cfg, args); err != nil {
		return err
	}

	var err error
	if cfg.RunID != "" {
		err = cancelByRunID(ctx, cfg, cfg.RunID, wOut)
	} else {
		execID := args[0]
		err = cancelExecution(ctx, cfg, execID, wOut)
	}
	return err
}

func validateCancel(cfg *config.SmartTestExecCancel, args []string) error {
	if len(args) > 1 {
		return errors.New("you can't specify more than a single execution name")
	}
	if len(args) == 0 {
		if cfg.RunID == "" {
			return errors.New("you must specify an execution name or provide a run ID")
		}
	} else if cfg.RunID != "" {
		return errors.New("you can't specify both an execution name and a run ID")
	}
	return nil
}

func cancelByRunID(ctx context.Context, cfg *config.SmartTestExecCancel, runID string,
	wOut io.Writer) error {
	// get all test executions
	txs, err := getTestExecutionsForRunID(ctx, cfg.SmartTest, runID)
	if err != nil {
		return err
	}

	for _, tx := range txs {
		err = cancelExecution(ctx, cfg, tx.ID, wOut)
		if err != nil {
			return fmt.Errorf("could not cancel test execution %q: %w", tx.ID, err)
		}
	}
	return nil
}

func cancelExecution(ctx context.Context, cfg *config.SmartTestExecCancel, execID string,
	wOut io.Writer) error {
	params := test_executions.NewCancelTestExecutionParams().
		WithContext(ctx).
		WithOrgName(cfg.Org).
		WithExecutionID(execID)
	_, err := cfg.Client.TestExecutions.CancelTestExecution(params, nil)
	if err != nil {
		return err
	}
	if cfg.OutputFormat == config.OutputFormatDefault {
		fmt.Fprintf(wOut, "Test execution %q canceled.\n", execID)
	}
	return nil
}
