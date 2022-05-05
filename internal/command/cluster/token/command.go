package token

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(cluster *config.Cluster) *cobra.Command {
	cfg := &config.ClusterToken{Cluster: cluster}

	cmd := &cobra.Command{
		Use:   "token",
		Short: "Manage auth tokens for cluster connections",
	}

	// Subcommands
	cmd.AddCommand(
		newCreate(cfg),
	)

	return cmd
}
