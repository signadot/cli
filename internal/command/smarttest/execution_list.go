package smarttest

import (
	"errors"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/go-sdk/client/test_executions"
	"github.com/spf13/cobra"
)

func newXList(tConfig *config.SmartTestExec) *cobra.Command {
	cfg := &config.SmartTestExecList{
		SmartTestExec: tConfig,
	}
	cmd := &cobra.Command{
		Use:   "list [filter-opts]",
		Short: "List test executions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return xList(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}
	cfg.AddFlags(cmd)
	return cmd
}

func xList(cfg *config.SmartTestExecList, wOut, wErr io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	pageSize := int64(200)
	orderDir := "desc"

	params := test_executions.NewQueryTestExecutionsParams().
		WithOrgName(cfg.Org).
		WithPageSize(&pageSize).
		WithOrderDir(&orderDir)
	if cfg.RunID != "" {
		params.WithRunID(&cfg.RunID)
	}
	if cfg.TestName != "" {
		params.WithTestName(&cfg.TestName)
	}
	if cfg.Sandbox != "" {
		params.WithTargetSandbox(&cfg.Sandbox)
	}
	if cfg.Repo != "" {
		params.WithRepo(&cfg.Repo)
	}
	if cfg.RepoPath != "" {
		params.WithRepoPath(&cfg.RepoPath)
	}
	if cfg.RepoCommitSHA != "" {
		params.WithRepoCommitSHA(&cfg.RepoCommitSHA)
	}
	if cfg.ExecutionPhase != "" {
		params.WithExecutionPhase(&cfg.ExecutionPhase)
	}
	if len(cfg.Labels) > 0 {
		params.WithLabel(cfg.Labels.ToQueryFilter())
	}
	result, err := cfg.Client.TestExecutions.QueryTestExecutions(params, nil)
	if err != nil {
		return err
	}
	if !result.IsSuccess() {
		return errors.New(result.Error())
	}
	txs := result.Payload
	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printTestExecutionsTable(wOut, txs)
	case config.OutputFormatJSON:
		return print.RawJSON(wOut, txs)
	case config.OutputFormatYAML:
		return print.RawYAML(wOut, txs)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
