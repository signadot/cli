package cluster

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/hack"
	"github.com/signadot/go-sdk/client/cluster"
	"github.com/spf13/cobra"
)

const connectMessage = `
Run the following commands against the desired cluster
to install Signadot Operator and populate the token:

kubectl create ns signadot
helm repo add signadot https://charts.signadot.com
helm install signadot-operator signadot/operator
kubectl -n signadot create secret generic cluster-agent --from-literal=token='%s'
`

func newAdd(cluster *config.Cluster) *cobra.Command {
	cfg := &config.ClusterAdd{Cluster: cluster}

	cmd := &cobra.Command{
		Use:   "add --name NAME",
		Short: "Add a Kubernetes cluster to Signadot",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return add(cfg, cmd.ErrOrStderr())
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func add(cfg *config.ClusterAdd, log io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	// Register the cluster name.
	params := cluster.NewAddClusterParams().
		WithOrgName(cfg.Org).WithClusterName(cfg.ClusterName)
	_, err := cfg.Client.Cluster.AddCluster(params, nil)
	if err != nil {
		return err
	}

	// Add the first token for this cluster.
	tokParams := cluster.NewCreateClusterTokenParams().
		WithOrgName(cfg.Org).WithClusterName(cfg.ClusterName)
	result, err := cfg.Client.Cluster.CreateClusterToken(tokParams, nil, hack.SendEmptyBody)
	if err != nil {
		return err
	}
	tokenValue := result.Payload.Token

	fmt.Fprintf(log, "Ready to connect cluster %q. A token has been created for the cluster.\n", cfg.ClusterName)
	fmt.Fprintf(log, connectMessage, tokenValue)

	return nil
}
