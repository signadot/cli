package signadot

import (
	"time"

	"github.com/signadot/cli/internal/tablewriter"
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
	Name        string
	Description string `maxLen:"25"`
	Cluster     string
	Created     time.Time
	Status      string
}

func (c *sandboxListCmd) run(cmd *cobra.Command, args []string) error {
	t, err := tablewriter.New[sandboxListRow](cmd.OutOrStdout())
	if err != nil {
		return err
	}

	// TODO: Fetch real data from the API.
	row := sandboxListRow{
		Name:        "test-foxish-mongo-2",
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
