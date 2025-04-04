package hostedtest

import (
	"errors"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/command/smarttest"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/client/test_executions"
	"github.com/signadot/go-sdk/models"
	libconncommon "github.com/signadot/libconnect/common"
	"github.com/spf13/cobra"
)

func newRun(tConfig *config.HostedTest) *cobra.Command {
	cfg := &config.HostedTestRun{
		HostedTest: tConfig,
	}
	cmd := &cobra.Command{
		Use:   "run <n>",
		Short: "Run a hosted test",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}
	cfg.AddFlags(cmd)
	return cmd
}

func run(cfg *config.HostedTestRun, wOut, wErr io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	// Handle single test execution
	if cfg.Cluster == "" {
		return fmt.Errorf("cluster flag is required for test execution")
	}

	testName := args[0]
	runID := libconncommon.GenerateRunID()

	txSpec := &models.TestExecutionSpec{
		Hosted: &models.HostedSpec{
			TestName: testName,
		},
		ExecutionContext: &models.TestExecutionContext{
			Cluster: cfg.Cluster,
			RunID:   runID,
		},
	}
	if cfg.Sandbox == "" && cfg.RouteGroup == "" {
		txSpec.ExecutionContext.AutoDiff = &models.TestExecutionAutoDiff{
			Enabled: false,
		}
	} else if cfg.Sandbox != "" && cfg.RouteGroup != "" {
		return fmt.Errorf("cannot specify both sandbox and route group")
	} else {
		txSpec.ExecutionContext.AutoDiff = &models.TestExecutionAutoDiff{
			Enabled: true,
		}
		rc := &models.JobRoutingContext{}
		txSpec.ExecutionContext.Routing = rc
		if cfg.Sandbox != "" {
			rc.Sandbox = cfg.Sandbox
		} else {
			rc.Routegroup = cfg.RouteGroup
		}
	}
	params := test_executions.NewCreateHostedTestExecutionParams().
		WithOrgName(cfg.Org).
		WithTestName(testName).
		WithData(&models.TestExecution{
			Spec: txSpec,
		})
	result, err := cfg.Client.TestExecutions.CreateHostedTestExecution(params, nil)
	if err != nil {
		return err
	}
	if !result.IsSuccess() {
		return errors.New(result.Error())
	}
	return smarttest.PrintTestExecution(cfg.OutputFormat, wOut, result.Payload)
}
