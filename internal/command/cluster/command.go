package cluster

import (
	"github.com/signadot/cli/internal/command/cluster/token"
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(root *config.Root) *cobra.Command {
	cfg := &config.Cluster{Root: root}
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Manage connections between your Kubernetes clusters and Signadot",
	}

	// Subcommands
	cmd.AddCommand(
		newConnect(cfg),
		token.New(cfg),
	)

	return cmd
}
