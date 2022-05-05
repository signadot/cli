package sandbox

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(root *config.Root) *cobra.Command {
	cfg := &config.Sandbox{Root: root}

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
	)

	return cmd
}
