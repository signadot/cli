package signadot

import (
	"time"

	"github.com/signadot/cli/internal/tablewriter"
	"github.com/spf13/cobra"
)

type sandboxGetCmd struct {
	*cobra.Command

	// Parent commands
	root    *RootCmd
	sandbox *sandboxCmd
}

func addSandboxGetCmd(sandbox *sandboxCmd) {
	c := &sandboxGetCmd{
		root:    sandbox.root,
		sandbox: sandbox,
	}
	c.Command = &cobra.Command{
		Use:   "get NAME",
		Short: "Get sandbox",
		Args:  cobra.ExactArgs(1),
		RunE:  c.run,
	}
	sandbox.AddCommand(c.Command)
}

func (c *sandboxGetCmd) run(cmd *cobra.Command, args []string) error {
	name := args[0]

	t, err := tablewriter.New[sandboxListRow](cmd.OutOrStdout())
	if err != nil {
		return err
	}

	// TODO: Fetch real data from the API.
	row := sandboxListRow{
		Name:        name,
		Description: "Sample sandbox created using Python SDK",
		Cluster:     "signadot-staging",
		Created:     time.Now(),
		Status:      "Ready",
	}
	if err := t.WriteRow(row); err != nil {
		return err
	}

	if err := t.Flush(); err != nil {
		return err
	}

	return nil
}
