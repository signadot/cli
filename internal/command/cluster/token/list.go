package token

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/go-sdk/client/cluster"
	"github.com/spf13/cobra"
)

func newList(cluster *config.ClusterToken) *cobra.Command {
	cfg := &config.ClusterTokenList{ClusterToken: cluster}

	cmd := &cobra.Command{
		Use:   "list --cluster CLUSTER",
		Short: "List auth tokens for the given cluster",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return list(cfg, cmd.OutOrStdout())
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func list(cfg *config.ClusterTokenList, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	params := cluster.NewListClusterTokensParams().
		WithOrgName(cfg.Org).WithClusterName(cfg.ClusterName)
	resp, err := cfg.Client.Cluster.ListClusterTokens(params, nil)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printTokenTable(out, resp.Payload)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
