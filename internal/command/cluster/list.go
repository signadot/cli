package cluster

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/go-sdk/client/cluster"
	"github.com/spf13/cobra"
)

func newList(cluster *config.Cluster) *cobra.Command {
	cfg := &config.ClusterList{Cluster: cluster}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List clusters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return list(cfg, cmd.OutOrStdout())
		},
	}

	return cmd
}

func list(cfg *config.ClusterList, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	resp, err := cfg.Client.Cluster.GetClusters(cluster.NewGetClustersParams().WithOrgName(cfg.Org), nil)
	if err != nil {
		return err
	}
	clusters := resp.Payload.Clusters

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return print.ClusterTable(out, clusters)
	case config.OutputFormatJSON:
		return print.RawJSON(out, clusters)
	case config.OutputFormatYAML:
		return print.RawYAML(out, clusters)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
