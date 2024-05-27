package artifact

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.Artifact{API: api}

	cmd := &cobra.Command{
		Use:   "artifact",
		Short: "Inspect and manipulate artifact",
	}

	// Subcommands
	cmd.AddCommand(
		newGet(cfg),
	)

	return cmd
}
