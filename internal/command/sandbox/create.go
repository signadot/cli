package sandbox

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func newCreate(sandbox *config.Sandbox) *cobra.Command {
	cfg := &config.SandboxCreate{Sandbox: sandbox}

	cmd := &cobra.Command{
		Use:   "create -f FILENAME",
		Short: "Create sandbox",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return create(cfg)
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func create(cfg *config.SandboxCreate) error {
	// TODO: Implement sandbox create.

	return nil
}
