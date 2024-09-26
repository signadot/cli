package test_exec

import (
	"errors"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/client/test_executions"
	"github.com/spf13/cobra"
)

func newGet(txConfig *config.TestExec) *cobra.Command {
	cfg := &config.TestExecGet{
		TestExec: txConfig,
	}
	cmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Get a test execution",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return get(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}
	cfg.AddFlags(cmd)
	return cmd
}

func get(cfg *config.TestExecGet, wOut, wErr io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	testName, execName, err := splitName(args[0])
	if err != nil {
		return err
	}

	params := test_executions.NewGetTestExecutionParams().WithOrgName(cfg.Org).
		WithTestName(testName).
		WithExecutionName(execName)
	result, err := cfg.Client.TestExecutions.GetTestExecution(params, nil)
	if err != nil {
		return err
	}
	if !result.IsSuccess() {
		return errors.New(result.Error())
	}
	return PrintTestExecution(cfg.OutputFormat, wOut, result.Payload)
}
