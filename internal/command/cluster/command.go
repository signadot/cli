package cluster

import (
	"github.com/signadot/cli/internal/command/cluster/token"
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.Cluster{API: api}
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Manage connections between your Kubernetes clusters and Signadot",
	}

	// Subcommands
	cmd.AddCommand(
		newAdd(cfg),
		newList(cfg),
		token.New(cfg),
	)

	return cmd
}
