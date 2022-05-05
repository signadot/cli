package token

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func newCreate(token *config.ClusterToken) *cobra.Command {
	cfg := &config.ClusterTokenCreate{ClusterToken: token}

	cmd := &cobra.Command{
		Use:   "create --cluster CLUSTER",
		Short: "Create an auth token for the given cluster",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return create(cfg)
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func create(cfg *config.ClusterTokenCreate) error {
	// TODO: Implement cluster token create.

	return nil
}
