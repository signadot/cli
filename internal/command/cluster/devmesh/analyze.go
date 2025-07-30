package devmesh

import (
	"fmt"
	"io"
	"strings"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/go-sdk/client/cluster"
	"github.com/spf13/cobra"
)

func newAnalyze(devmesh *config.ClusterDevMesh) *cobra.Command {
	cfg := &config.ClusterDevMeshAnalyze{ClusterDevMesh: devmesh}

	cmd := &cobra.Command{
		Use:   "analyze --cluster CLUSTER",
		Short: "Analyze DevMesh enabled workloads",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return analyze(cfg, cmd.OutOrStdout())
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func analyze(cfg *config.ClusterDevMeshAnalyze, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	params := cluster.NewClusterDevmeshAnalyzeParams().
		WithOrgName(cfg.Org).WithClusterName(cfg.ClusterName)
	if cfg.Namespace != "" {
		params = params.WithNamespace(&cfg.Namespace)
	}
	if cfg.Status != "" {
		sts := strings.ToLower(cfg.Status)
		params = params.WithStatus(&sts)
	}

	resp, err := cfg.Client.Cluster.ClusterDevmeshAnalyze(params, nil)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printDevMeshAnalysisTable(out, resp.Payload)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
