package sandbox

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
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

func list(cfg *config.SandboxList, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	resp, err := cfg.Client.Sandboxes.GetSandboxes(sandboxes.NewGetSandboxesParams().WithOrgName(cfg.Org), nil)
	if err != nil {
		return err
	}
	sbs := resp.Payload.Sandboxes

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return print.SandboxTable(out, sbs)
	case config.OutputFormatJSON:
		return print.RawJSON(out, sbs)
	case config.OutputFormatYAML:
		return print.RawYAML(out, sbs)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
