package smarttest

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func newExecution(tConfig *config.SmartTest) *cobra.Command {
	cfg := &config.SmartTestExec{
		SmartTest: tConfig,
	}
	cmd := &cobra.Command{
		Use:     "execution",
		Aliases: []string{"x"},
		Short:   "Work with smart test executions",
	}
	get := newGet(cfg)
	list := newList(cfg)
	cancel := newCancel(cfg)
	cmd.AddCommand(get)
	cmd.AddCommand(list)
	cmd.AddCommand(cancel)
	return cmd
}
