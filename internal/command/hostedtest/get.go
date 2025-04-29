package hostedtest

import (
	"errors"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/client/tests"
	"github.com/spf13/cobra"
)

func newGet(tstConfig *config.HostedTest) *cobra.Command {
	cfg := &config.HostedTestGet{
		HostedTest: tstConfig,
	}
	cmd := &cobra.Command{
		Use:   "get <n>",
		Short: "Get a hosted test",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return get(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}
	cfg.AddFlags(cmd)
	return cmd
}

func get(cfg *config.HostedTestGet, wOut, wErr io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	name := args[0]
	params := tests.NewGetTestParams().WithOrgName(cfg.Org).WithTestName(name)
	result, err := cfg.Client.Tests.GetTest(params, nil)
	if err != nil {
		return err
	}
	if !result.IsSuccess() {
		return errors.New(result.Error())
	}
	return printTest(cfg.OutputFormat, wOut, result.Payload)
}
