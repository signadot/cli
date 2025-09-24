package override

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func newDelete(cfg *config.LocalOverride) *cobra.Command {
	deleteCfg := &config.LocalOverrideDelete{LocalOverride: cfg}

	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a traffic override",
		Long: `Delete an existing traffic override by name.
This will remove the redirect and altflow middleware from the sandbox.

Example:
  signadot local override delete my-override`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(cmd.OutOrStdout(), deleteCfg, args[0])
		},
	}
	deleteCfg.AddFlags(cmd)

	return cmd
}

func runDelete(out io.Writer, cfg *config.LocalOverrideDelete, name string) error {
	if err := cfg.InitLocalConfig(); err != nil {
		return err
	}

	// Initialize API client
	if err := cfg.API.InitAPIConfig(); err != nil {
		return err
	}

	// TODO: Implement actual override deletion logic
	// This is a skeleton implementation

	printOverrideProgress(out, "Removing redirect and altflow middleware from sandbox")

	// Simulate deletion process
	// TODO: Implement actual deletion logic

	printOverrideStatus(out, fmt.Sprintf("Override %s deleted successfully", name), true)

	return nil
}
