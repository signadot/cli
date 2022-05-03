package signadot

import "github.com/spf13/cobra"

type clusterTokenCmd struct {
	*cobra.Command

	// Parent commands
	root    *RootCmd
	cluster *clusterCmd
}

func addClusterTokenCmd(cluster *clusterCmd) {
	c := &clusterTokenCmd{
		root:    cluster.root,
		cluster: cluster,
	}
	c.Command = &cobra.Command{
		Use:   "token",
		Short: "Manage auth tokens for cluster connections",
	}

	// Subcommands
	addClusterTokenCreateCmd(c)

	cluster.AddCommand(c.Command)
}
