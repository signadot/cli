package sandbox

import (
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/spf13/cobra"
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
	Status      string `sdtab:"STATUS"`
}

func list(cfg *config.SandboxList, out io.Writer) error {
	resp, err := cfg.Client.Sandboxes.GetSandboxes(sandboxes.NewGetSandboxesParams().WithOrgName(cfg.Org), cfg.AuthInfo)
	if err != nil {
		return err
	}
	sbs := resp.Payload.Sandboxes

	t := sdtab.New[tableRow](out)
	t.AddHeader()
	for _, sbinfo := range sbs {
		row := tableRow{
			Name:        sbinfo.Name,
			Description: sbinfo.Description,
			Cluster:     sbinfo.ClusterName,
			Created:     sbinfo.CreatedAt,
			Status:      "Ready",
		}
		t.AddRow(row)
	}
	if err := t.Flush(); err != nil {
		return err
	}

	return nil
}
