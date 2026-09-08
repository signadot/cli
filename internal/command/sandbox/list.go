package sandbox

import (
	"context"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/sdkclient"
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

func list(cfg *config.SandboxList, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	sandboxes, err := sdkclient.ListAllSandboxes(context.Background(), cfg.Client, cfg.Org)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printSandboxTable(out, sandboxes)
	case config.OutputFormatJSON:
		return print.RawJSON(out, sandboxes)
	case config.OutputFormatYAML:
		return print.RawYAML(out, sandboxes)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
