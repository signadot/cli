package cluster

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/hack"
	"github.com/signadot/cli/internal/shallow"
	"github.com/signadot/go-sdk/client/cluster"
	"github.com/signadot/go-sdk/models"
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

func newConnect(cluster *config.Cluster) *cobra.Command {
	cfg := &config.ClusterConnect{Cluster: cluster}

	cmd := &cobra.Command{
		Use:   "connect --name NAME",
		Short: "Register a Kubernetes cluster with Signadot",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return connect(cfg, cmd.OutOrStdout())
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func connect(cfg *config.ClusterConnect, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	// Register the cluster name.
	req := &models.ConnectClusterRequest{
		Name: shallow.Copy(cfg.ClusterName),
	}
	params := cluster.NewConnectClusterParams().
		WithOrgName(cfg.Org).WithData(req)
	_, err := cfg.Client.Cluster.ConnectCluster(params, nil)
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

	fmt.Fprintf(out, "Ready to connect cluster %q. A token has been created for the cluster.\n", cfg.ClusterName)
	fmt.Fprintf(out, connectMessage, tokenValue)

	return nil
}
