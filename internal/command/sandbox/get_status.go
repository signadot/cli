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

func newGetStatus(sandbox *config.Sandbox) *cobra.Command {
	cfg := &config.SandboxGetStatus{Sandbox: sandbox}

	cmd := &cobra.Command{
		Use:   "get-status NAME",
		Short: "Get sandbox status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return getStatus(cfg, cmd.OutOrStdout(), args[0])
		},
	}

	return cmd
}

func getStatus(cfg *config.SandboxGetStatus, out io.Writer, name string) error {
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

	// Get sandbox status.
	params := sandboxes.NewGetSandboxStatusByIDParams().WithOrgName(cfg.Org).WithSandboxID(sb.ID)
	statusResp, err := cfg.Client.Sandboxes.GetSandboxStatusByID(params, nil)
	if err != nil {
		return err
	}
	status := statusResp.Payload.Status

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		ready := "Not Ready"
		if status.Ready {
			ready = "Ready"
		}
		fmt.Fprintf(out, "%s: %s\n", ready, status.Message)
		return nil
	case config.OutputFormatJSON:
		return print.RawJSON(out, status)
	case config.OutputFormatYAML:
		return print.RawYAML(out, status)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
