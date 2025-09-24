package override

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/spf13/cobra"
)

func newList(cfg *config.LocalOverride) *cobra.Command {
	listCfg := &config.LocalOverrideList{LocalOverride: cfg}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List active traffic overrides",
		Long: `List all active traffic overrides for the specified cluster.
Shows the name, target sandbox, and what it's overridden by.

Example:
  signadot local override list --cluster=signadot-staging`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.OutOrStdout(), listCfg)
		},
	}
	listCfg.AddFlags(cmd)

	return cmd
}

func runList(out io.Writer, cfg *config.LocalOverrideList) error {
	if err := cfg.InitLocalConfig(); err != nil {
		return err
	}

	// Initialize API client
	if err := cfg.API.InitAPIConfig(); err != nil {
		return err
	}

	// TODO: Implement actual override listing logic
	// This is a skeleton implementation

	// Create sample overrides for demonstration
	overrides := []*Override{
		{
			Name:      "my-override",
			Sandbox:   "test",
			Target:    "localhost:5000",
			Cluster:   cfg.Cluster,
			CreatedAt: "2024-01-15T10:30:00Z",
			Status:    "active",
			Detached:  false,
		},
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printOverrideTable(out, overrides)
	case config.OutputFormatJSON:
		return print.RawJSON(out, overrides)
	case config.OutputFormatYAML:
		return print.RawYAML(out, overrides)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
