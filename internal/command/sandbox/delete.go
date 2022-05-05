package sandbox

import (
	"errors"

	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func newDelete(sandbox *config.Sandbox) *cobra.Command {
	cfg := &config.SandboxDelete{Sandbox: sandbox}

	cmd := &cobra.Command{
		Use:   "delete { -f FILENAME | NAME }",
		Short: "Delete sandbox",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return delete(cfg, args)
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func delete(cfg *config.SandboxDelete, args []string) error {
	if cfg.Filename == "" && len(args) == 0 {
		return errors.New("must specify either filename or sandbox name")
	}
	if cfg.Filename != "" && len(args) > 0 {
		return errors.New("can't specify both filename and sandbox name")
	}

	// TODO: Implement sandbox delete.

	return nil
}
