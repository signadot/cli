package sandbox

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

func newList(sandbox *config.Sandbox) *cobra.Command {
	cfg := &config.SandboxList{Sandbox: sandbox}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sandboxes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return list(cfg, cmd.OutOrStdout())
		},
	}

	return cmd
}

type tableRow struct {
	Name        string `sdtab:"NAME"`
	Description string `sdtab:"DESCRIPTION,trunc"`
	Cluster     string `sdtab:"CLUSTER"`
	Created     string `sdtab:"CREATED"`
}

func list(cfg *config.SandboxList, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	resp, err := cfg.Client.Sandboxes.GetSandboxes(sandboxes.NewGetSandboxesParams().WithOrgName(cfg.Org), nil)
	if err != nil {
		return err
	}
	sbs := resp.Payload.Sandboxes

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		t := sdtab.New[tableRow](out)
		t.AddHeader()
		for _, sbinfo := range sbs {
			row := tableRow{
				Name:        sbinfo.Name,
				Description: sbinfo.Description,
				Cluster:     sbinfo.ClusterName,
				Created:     sbinfo.CreatedAt,
			}
			t.AddRow(row)
		}
		if err := t.Flush(); err != nil {
			return err
		}
	case config.OutputFormatJSON:
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(sbs); err != nil {
			return err
		}
	case config.OutputFormatYAML:
		data, err := yaml.Marshal(sbs)
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
