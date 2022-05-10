package cluster

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/client/cluster"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
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

type tableRow struct {
	Name    string `sdtab:"NAME"`
	Created string `sdtab:"CREATED"`
	Version string `sdtab:"OPERATOR VERSION"`
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
		t := sdtab.New[tableRow](out)
		t.AddHeader()
		for _, cluster := range clusters {
			row := tableRow{
				Name:    cluster.Name,
				Created: cluster.CreatedAt,
				Version: cluster.OperatorVersion,
			}
			t.AddRow(row)
		}
		if err := t.Flush(); err != nil {
			return err
		}
	case config.OutputFormatJSON:
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(clusters); err != nil {
			return err
		}
	case config.OutputFormatYAML:
		data, err := yaml.Marshal(clusters)
		if err != nil {
			return err
		}
		if _, err := out.Write(data); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}

	return nil
}
