package config

import "github.com/spf13/cobra"

type ClusterDevMesh struct {
	*Cluster
}

type ClusterDevMeshAnalyze struct {
	*ClusterDevMesh

	// Flags
	ClusterName string
	Namespace   string
	Status      string
}

func (c *ClusterDevMeshAnalyze) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.ClusterName, "cluster", "", "name of the cluster to analyze")
	cmd.MarkFlagRequired("cluster")
	cmd.Flags().StringVar(&c.Namespace, "namespace", "", "filter results by namespace (comma separated list)")
	cmd.Flags().StringVar(&c.Status, "status", "", "filter results by target status (comma separated list)")
}
