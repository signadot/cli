package token

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/hack"
	"github.com/signadot/go-sdk/client/cluster"
	"github.com/spf13/cobra"
)

func newCreate(token *config.ClusterToken) *cobra.Command {
	cfg := &config.ClusterTokenCreate{ClusterToken: token}

	cmd := &cobra.Command{
		Use:   "create --cluster CLUSTER",
		Short: "Create an auth token for the given cluster",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return create(cfg, cmd.OutOrStdout())
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func create(cfg *config.ClusterTokenCreate, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	params := cluster.NewCreateClusterTokenParams().
		WithOrgName(cfg.Org).WithClusterName(cfg.ClusterName)
	result, err := cfg.Client.Cluster.CreateClusterToken(params, cfg.AuthInfo, hack.SendEmptyBody)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, result.Payload.Token)

	return nil
}
