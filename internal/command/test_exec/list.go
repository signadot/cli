package test_exec

import (
	"errors"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/client/test_executions"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newList(txConfig *config.TestExec) *cobra.Command {
	cfg := &config.TestExecList{
		TestExec: txConfig,
	}
	cmd := &cobra.Command{
		Use:   "list <test-name>",
		Short: "List test executions for a test",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return list(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}
	cfg.AddFlags(cmd)
	return cmd
}

func list(cfg *config.TestExecList, wOut, wErr io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	testName := args[0]
	params := test_executions.NewListTestExecutionsParams().
		WithOrgName(cfg.Org).
		WithTestName(testName)
	result, err := cfg.Client.TestExecutions.ListTestExecutions(params, nil)
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
	return nil
}

type testExecRow struct {
	Name      string `sdtab:"NAME"`
	Phase     string `sdtab:"PHASE"`
	CreatedAt string `sdtab:"CREATED"`
}

func printTestExecutionsTable(w io.Writer, txs []*models.TestExecution) error {
	tab := sdtab.New[testExecRow](w)
	tab.AddHeader()
	for _, tx := range txs {
		tab.AddRow(testExecRow{
			Name:      tx.Name,
			CreatedAt: tx.CreatedAt,
			Phase:     tx.Status.Phase,
		})
	}
	return tab.Flush()
}
