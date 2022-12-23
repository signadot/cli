package routegroup

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.RouteGroup{API: api}

	cmd := &cobra.Command{
		Use:   "routegroup",
		Short: "Inspect and manipulate routegroups",
	}

	// Subcommands
	cmd.AddCommand(
		newGet(cfg),
		newList(cfg),
		newApply(cfg),
		newDelete(cfg),
	)

	return cmd
}
