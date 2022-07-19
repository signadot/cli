package token

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/hack"
	"github.com/signadot/go-sdk/client/cluster"
	"github.com/spf13/cobra"
)

func newDelete(token *config.ClusterToken) *cobra.Command {
	cfg := &config.ClusterTokenDelete{ClusterToken: token}

	cmd := &cobra.Command{
		Use:   "delete --cluster=CLUSTER TOKEN_ID",
		Short: "Delete an auth token for the given cluster",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return delete(cfg, cmd.ErrOrStderr(), args[0])
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func delete(cfg *config.ClusterTokenDelete, log io.Writer, id string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	params := cluster.NewDeleteClusterTokenParams().
		WithOrgName(cfg.Org).WithClusterName(cfg.ClusterName).WithTokenID(id)
	_, err := cfg.Client.Cluster.DeleteClusterToken(params, nil, hack.SendEmptyBody)
	if err != nil {
		return err
	}

	fmt.Fprintf(log, "Deleted token ID %q for cluster %q.\n", id, cfg.ClusterName)

	return nil
}
