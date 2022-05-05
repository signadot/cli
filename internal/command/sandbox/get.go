package sandbox

import (
	"io"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/spf13/cobra"
)

func newGet(sandbox *config.Sandbox) *cobra.Command {
	cfg := &config.SandboxGet{Sandbox: sandbox}

	cmd := &cobra.Command{
		Use:   "get NAME",
		Short: "Get sandbox",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return get(cfg, cmd.OutOrStdout(), args[0])
		},
	}

	return cmd
}

func get(cfg *config.SandboxGet, out io.Writer, name string) error {
	// TODO: Fetch real data from the API.

	t := sdtab.New[tableRow](out)
	t.AddHeader()
	row := tableRow{
		Name:        name,
		Description: "Sample sandbox created using Python SDK",
		Cluster:     "signadot-staging",
		Created:     time.Now().String(),
		Status:      "Ready",
	}
	t.AddRow(row)
	if err := t.Flush(); err != nil {
		return err
	}

	return nil
}
