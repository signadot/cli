package config

import "github.com/spf13/cobra"

type Cluster struct {
	*Api
}

type ClusterAdd struct {
	*Cluster

	// Flags
	ClusterName string
}

func (c *ClusterAdd) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.ClusterName, "name", "", "assign a name for this cluster as it will appear within Signadot")
	cmd.MarkFlagRequired("name")
}

type ClusterList struct {
	*Cluster
}
