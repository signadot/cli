package signadot

import (
	"github.com/spf13/cobra"
)

type clusterConnectCmd struct {
	*cobra.Command

	// Parent commands
	root    *RootCmd
	cluster *clusterCmd

	// Flags
	name string
}

func addClusterConnectCmd(cluster *clusterCmd) {
	c := &clusterConnectCmd{
		root:    cluster.root,
		cluster: cluster,
	}
	c.Command = &cobra.Command{
		Use:   "connect --name NAME",
		Short: "Register a Kubernetes cluster with Signadot",
		Args:  cobra.NoArgs,
		RunE:  c.run,
	}

	c.Flags().StringVar(&c.name, "name", "", "assign a name for this cluster as it will appear within Signadot")
	c.MarkFlagRequired("name")

	cluster.AddCommand(c.Command)
}

func (c *clusterConnectCmd) run(cmd *cobra.Command, args []string) error {
	// TODO: Implement sandbox create.

	return nil
}
