package test

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/signadot/cli/internal/command/test_exec"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/repoconfig"
	"github.com/signadot/go-sdk/client/test_executions"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newRun(tConfig *config.Test) *cobra.Command {
	cfg := &config.TestRun{
		Test: tConfig,
	}
	cmd := &cobra.Command{
		Use:   "run [name]",
		Short: "Run a test",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}
	cfg.AddFlags(cmd)
	return cmd
}

func run(cfg *config.TestRun, wOut, wErr io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	// If no arguments provided, try to read from .signadot/config
	if len(args) == 0 {
		// Get current directory
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Load config
		config, err := repoconfig.LoadConfig(cwd)
		if err != nil {
			return fmt.Errorf("failed to load .signadot/config.yaml: %w", err)
		}

		// Find test files
		testFiles, err := repoconfig.FindTestFiles(cwd, config)
		if err != nil {
			return fmt.Errorf("failed to find test files: %w", err)
		}

		// Print test files
		fmt.Fprintf(wOut, "Found %d test files:\n", len(testFiles))
		for _, tf := range testFiles {
			fmt.Fprintf(wOut, "  %s\n", tf.Path)
			if len(tf.Labels) > 0 {
				fmt.Fprintf(wOut, "    Labels:\n")
				for k, v := range tf.Labels {
					fmt.Fprintf(wOut, "      %s=%s\n", k, v)
				}
			}
		}
		return nil
	}

	// Handle single test execution
	if cfg.Cluster == "" {
		return fmt.Errorf("cluster flag is required for test execution")
	}

	name := args[0]
	txSpec := &models.TestExecutionSpec{
		Test: name,
		ExecutionContext: &models.TestExecutionContext{
			Cluster: cfg.Cluster,
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
	params := test_executions.NewCreateTestExecutionParams().
		WithOrgName(cfg.Org).
		WithTestName(name).
		WithData(txSpec)
	result, err := cfg.Client.TestExecutions.CreateTestExecution(params, nil)
	if err != nil {
		return err
	}
	if !result.IsSuccess() {
		return errors.New(result.Error())
	}
	return test_exec.PrintTestExecution(cfg.OutputFormat, wOut, result.Payload)
}
