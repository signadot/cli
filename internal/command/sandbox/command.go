package sandbox

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.Api) *cobra.Command {
	cfg := &config.Sandbox{Api: api}

	cmd := &cobra.Command{
		Use:   "sandbox",
		Short: "Inspect and manipulate sandboxes",
	}

	// Subcommands
	cmd.AddCommand(
		newGet(cfg),
		newList(cfg),
		newCreate(cfg),
		newDelete(cfg),
		newGetStatus(cfg),
	)

	return cmd
}
