package test

import (
	"errors"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/client/tests"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newList(tstConfig *config.Test) *cobra.Command {
	cfg := &config.TestList{
		Test: tstConfig,
	}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list tests",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return list(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}
	cfg.AddFlags(cmd)
	return cmd
}

func list(cfg *config.TestList, wOut, wErr io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	params := tests.NewListTestsParams().WithOrgName(cfg.Org)
	result, err := cfg.Client.Tests.ListTests(params, nil)
	if err != nil {
		return err
	}
	if !result.IsSuccess() {
		return errors.New(result.Error())
	}
	ts := result.Payload
	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printTestTable(wOut, ts)
	case config.OutputFormatJSON:
		return print.RawJSON(wOut, ts)
	case config.OutputFormatYAML:
		return print.RawYAML(wOut, ts)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
	return nil
}

type testRow struct {
	Name      string `sdtab:"NAME"`
	CreatedAt string `sdtab:"CREATED"`
}

func printTestTable(w io.Writer, ts []*models.Test) error {
	tab := sdtab.New[testRow](w)
	tab.AddHeader()
	for _, t := range ts {
		tab.AddRow(testRow{
			Name:      t.Name,
			CreatedAt: t.CreatedAt,
		})
	}
	return tab.Flush()
}
