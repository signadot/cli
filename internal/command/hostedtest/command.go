package hostedtest

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.HostedTest{API: api}
	cmd := &cobra.Command{
		Use:     "hosted-test",
		Short:   "Signadot hosted tests",
		Aliases: []string{"ht"},
		Hidden:  true,
	}

	apply := newApply(cfg)
	get := newGet(cfg)
	lst := newList(cfg)
	del := newDelete(cfg)
	run := newRun(cfg)

	// Subcommands
	cmd.AddCommand(run, get, lst, del, apply)

	return cmd
}
