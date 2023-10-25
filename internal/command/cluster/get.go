package cluster

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/go-sdk/client/cluster"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newGet(cluster *config.Cluster) *cobra.Command {
	cfg := &config.ClusterGet{Cluster: cluster}

	cmd := &cobra.Command{
		Use:   "get",
		Short: "get a cluster",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return get(cfg, cmd.OutOrStdout(), args)
		},
	}

	return cmd
}

func get(cfg *config.ClusterGet, out io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	params := cluster.NewGetClusterParams().
		WithOrgName(cfg.Org).
		WithClusterName(args[0])
	resp, err := cfg.Client.Cluster.GetCluster(params, nil)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printClusterTable(out, []*models.Cluster{resp.Payload})
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
