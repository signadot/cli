package signadot

import (
	"github.com/signadot/cli/internal/auth"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/client"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/spf13/cobra"
)

type sandboxListCmd struct {
	*cobra.Command

	// Parent commands
	root    *RootCmd
	sandbox *sandboxCmd
}

func addSandboxListCmd(sandbox *sandboxCmd) {
	c := &sandboxListCmd{
		root:    sandbox.root,
		sandbox: sandbox,
	}
	c.Command = &cobra.Command{
		Use:   "list",
		Short: "List sandboxes",
		Args:  cobra.NoArgs,
		RunE:  c.run,
	}
	sandbox.AddCommand(c.Command)
}

type sandboxListRow struct {
	Name        string `sdtab:"NAME,10,-"`
	Description string `sdtab:"DESCRIPTION,15,0"`
	Cluster     string `sdtab:"CLUSTER,10,0"`
	Created     string `sdtab:"CREATED,10,0"`
	Status      string `sdtab:"STATUS,5,-"`
}

func (c *sandboxListCmd) run(cmd *cobra.Command, args []string) error {
	t := sdtab.New[sandboxListRow](cmd.OutOrStdout())
	if err := t.WriteHeader(); err != nil {
		return err
	}

	d := client.Default
	// TODO: how to handle org?
	resp, err := d.Sandboxes.GetSandboxes(sandboxes.NewGetSandboxesParams().WithOrgName("signadot"), auth.Authenticator())
	if err != nil {
		return err
	}
	sbs := resp.Payload.Sandboxes
	for _, sbinfo := range sbs {
		row := sandboxListRow{
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
