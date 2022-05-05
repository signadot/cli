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
	Name        string `sdtab:"NAME,10,-"`
	Description string `sdtab:"DESCRIPTION,15,0"`
	Cluster     string `sdtab:"CLUSTER,10,0"`
	Created     string `sdtab:"CREATED,10,0"`
	Status      string `sdtab:"STATUS,5,-"`
}

func list(cfg *config.SandboxList, out io.Writer) error {
	t := sdtab.New[tableRow](out)
	if err := t.WriteHeader(); err != nil {
		return err
	}
	resp, err := cfg.Client.Sandboxes.GetSandboxes(sandboxes.NewGetSandboxesParams().WithOrgName(cfg.Org), cfg.AuthInfo)
	if err != nil {
		return err
	}
	sbs := resp.Payload.Sandboxes
	for _, sbinfo := range sbs {
		row := tableRow{
			Name:        sbinfo.Name,
			Description: sbinfo.Description,
			Cluster:     sbinfo.ClusterName,
			Created:     sbinfo.CreatedAt,
			Status:      "Ready",
		}

		if err := t.WriteRow(row); err != nil {
			return err
		}
	}

	if err := t.Flush(); err != nil {
		return err
	}

	return nil
}
