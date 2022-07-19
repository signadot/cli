package config

import "github.com/spf13/cobra"

type ClusterToken struct {
	*Cluster
}

type ClusterTokenCreate struct {
	*ClusterToken

	// Flags
	ClusterName string
}

func (c *ClusterTokenCreate) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.ClusterName, "cluster", "", "name of the cluster as it's known within Signadot")
	cmd.MarkFlagRequired("cluster")
}

type ClusterTokenList struct {
	*ClusterToken

	// Flags
	ClusterName string
}

func (c *ClusterTokenList) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.ClusterName, "cluster", "", "name of the cluster as it's known within Signadot")
	cmd.MarkFlagRequired("cluster")
}

type ClusterTokenDelete struct {
	*ClusterToken

	// Flags
	ClusterName string
}

func (c *ClusterTokenDelete) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.ClusterName, "cluster", "", "name of the cluster as it's known within Signadot")
	cmd.MarkFlagRequired("cluster")
}
