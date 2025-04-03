package smarttest

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.SmartTest{API: api}
	cmd := &cobra.Command{
		Use:     "smart-test",
		Short:   "Signadot smart tests",
		Aliases: []string{"st"},
	}

	run := newRun(cfg)
	get := newGet(cfg)
	list := newList(cfg)
	cancel := newCancel(cfg)

	// Subcommands
	cmd.AddCommand(run)
	cmd.AddCommand(get)
	cmd.AddCommand(list)
	cmd.AddCommand(cancel)

	return cmd
}
