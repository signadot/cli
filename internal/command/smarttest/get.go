package smarttest

import (
	"errors"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/client/test_executions"
	"github.com/spf13/cobra"
)

func newGet(tConfig *config.SmartTest) *cobra.Command {
	cfg := &config.SmartTestGet{
		SmartTest: tConfig,
	}
	cmd := &cobra.Command{
		Use:     "get-execution <name>",
		Short:   "Get a test execution (with alias getx)",
		Aliases: []string{"getx"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return get(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}
	cfg.AddFlags(cmd)
	return cmd
}

func get(cfg *config.SmartTestGet, wOut, wErr io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	execName := args[0]

	params := test_executions.NewGetTestExecutionParams().WithOrgName(cfg.Org).
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
