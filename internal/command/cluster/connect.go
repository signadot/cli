package cluster

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func newConnect(cluster *config.Cluster) *cobra.Command {
	cfg := &config.ClusterConnect{Cluster: cluster}

	cmd := &cobra.Command{
		Use:   "connect --name NAME",
		Short: "Register a Kubernetes cluster with Signadot",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return connect(cfg)
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func connect(cfg *config.ClusterConnect) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	// TODO: Implement cluster connect.

	return nil
}
