package secret

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.Secret{API: api}

	cmd := &cobra.Command{
		Use:     "secret",
		Short:   "Manage org-level secrets",
		Aliases: []string{"secrets"},
	}

	cmd.AddCommand(
		newCreate(cfg),
		newUpdate(cfg),
		newGet(cfg),
		newList(cfg),
		newDelete(cfg),
	)

	return cmd
}
