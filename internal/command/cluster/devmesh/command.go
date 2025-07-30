package devmesh

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(cluster *config.Cluster) *cobra.Command {
	cfg := &config.ClusterDevMesh{Cluster: cluster}

	cmd := &cobra.Command{
		Use:   "devmesh",
		Short: "Manage DevMesh enabled workloads",
	}

	// Subcommands
	cmd.AddCommand(
		newAnalyze(cfg),
	)

	return cmd
}
