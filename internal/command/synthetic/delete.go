package synthetic

import (
	"errors"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/client/tests"

	"github.com/spf13/cobra"
)

func newDelete(tstConfig *config.Synthetic) *cobra.Command {
	cfg := &config.SyntheticDelete{
		Synthetic: tstConfig,
	}
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a synthetic test",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deleteTest(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}
	cfg.AddFlags(cmd)
	return cmd
}

func deleteTest(cfg *config.SyntheticDelete, wOut, wErr io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	name := args[0]
	params := tests.NewDeleteTestParams().WithOrgName(cfg.Org).WithTestName(name)
	result, err := cfg.Client.Tests.DeleteTest(params, nil)
	if err != nil {
		return err
	}
	if !result.IsSuccess() {
		return errors.New(result.Error())
	}
	fmt.Fprintf(wOut, "test %q deleted.\n", name)
	return nil
}
