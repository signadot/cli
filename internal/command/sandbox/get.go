package sandbox

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
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
	// TODO: Use GetSandboxByName when it's available.
	resp, err := cfg.Client.Sandboxes.GetSandboxes(sandboxes.NewGetSandboxesParams().WithOrgName(cfg.Org), cfg.AuthInfo)
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
		t := sdtab.New[*sbRow](out, &sbRow{})
		t.AddHeader()
		row := &sbRow{SandboxInfo: *sb, status: "Ready"}
		t.AddRow(row)
		if err := t.Flush(); err != nil {
			return err
		}
	case config.OutputFormatJSON:
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(sb); err != nil {
			return err
		}
	case config.OutputFormatYAML:
		data, err := yaml.Marshal(sb)
		if err != nil {
			return err
		}
		if _, err := out.Write(data); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}

	return nil
}
