package override

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
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

	sandboxes, err := getSandboxes(cfg)
	if err != nil {
		return err
	}

	overrides, err := getOverridesFromSandboxes(sandboxes)
	if err != nil {
		return err
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

func getSandboxes(cfg *config.LocalOverrideList) ([]*models.Sandbox, error) {
	resp, err := cfg.Client.Sandboxes.
		ListSandboxes(sandboxes.NewListSandboxesParams().
			WithOrgName(cfg.Org), nil)
	if err != nil {
		return nil, err
	}

	return resp.Payload, nil
}

func getOverridesFromSandboxes(sandboxes []*models.Sandbox) ([]*Override, error) {
	overrides := make([]*Override, 0)
	for _, sandbox := range sandboxes {

		if sandbox.Spec.Routing == nil {
			continue
		}

		if sandbox.Spec.Routing.Forwards == nil {
			continue
		}

		for _, override := range sandbox.Spec.Routing.Forwards {
			overrides = append(overrides, &Override{
				Name:      override.Name,
				Sandbox:   sandbox.Name,
				ToLocal:   override.ToLocal,
				CreatedAt: sandbox.CreatedAt,
			})
		}
	}

	return overrides, nil
}
