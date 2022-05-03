package signadot

import (
	"github.com/spf13/cobra"
)

type clusterTokenCreateCmd struct {
	*cobra.Command

	// Parent commands
	root         *RootCmd
	cluster      *clusterCmd
	clusterToken *clusterTokenCmd

	// Flags
	clusterName string
}

func addClusterTokenCreateCmd(token *clusterTokenCmd) {
	c := &clusterTokenCreateCmd{
		root:         token.cluster.root,
		cluster:      token.cluster,
		clusterToken: token,
	}
	c.Command = &cobra.Command{
		Use:   "create --cluster CLUSTER",
		Short: "Create an auth token for the given cluster",
		Args:  cobra.NoArgs,
		RunE:  c.run,
	}

	c.Flags().StringVar(&c.clusterName, "cluster", "", "name of the cluster as it's known within Signadot")
	c.MarkFlagRequired("cluster")

	token.AddCommand(c.Command)
}

func (c *clusterTokenCreateCmd) run(cmd *cobra.Command, args []string) error {
	// TODO: Implement sandbox create.

	return nil
}
