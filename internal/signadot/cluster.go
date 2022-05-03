package signadot

import "github.com/spf13/cobra"

type clusterCmd struct {
	*cobra.Command

	// Parent commands
	root *RootCmd
}

func addClusterCmd(root *RootCmd) {
	c := &clusterCmd{root: root}
	c.Command = &cobra.Command{
		Use:   "cluster",
		Short: "Manage connections between your Kubernetes clusters and Signadot",
	}

	// Subcommands
	addClusterConnectCmd(c)
	addClusterTokenCmd(c)

	root.AddCommand(c.Command)
}
