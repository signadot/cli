package sandbox

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
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
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	// TODO: Use GetSandboxByName when it's available.
	resp, err := cfg.Client.Sandboxes.GetSandboxes(sandboxes.NewGetSandboxesParams().WithOrgName(cfg.Org), nil)
	if err != nil {
		return err
	}
	var sb *models.SandboxInfo
	for _, sbinfo := range resp.Payload.Sandboxes {
		if sbinfo.Name == name {
			sb = sbinfo
			break
		}
	}
	if sb == nil {
		return fmt.Errorf("Sandbox %q not found", name)
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return print.SandboxDetails(cfg, out, sb)
	case config.OutputFormatJSON:
		return print.RawJSON(out, sb)
	case config.OutputFormatYAML:
		return print.RawYAML(out, sb)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
