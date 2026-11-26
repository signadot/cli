package override

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/builder"
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
  signadot local override list`,
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
	// The v2 list endpoint returns a paginated {items,nextCursor,hasMore}
	// envelope (the go-sdk defaults the opt-in to "pagination"). Walk every
	// page so overrides are computed over all sandboxes, not just the first page.
	var items []*models.Sandbox
	var cursor *string
	for {
		params := sandboxes.NewListSandboxesParams().WithOrgName(cfg.Org)
		if cursor != nil {
			params = params.WithCursor(cursor)
		}
		resp, err := cfg.Client.Sandboxes.ListSandboxes(params, nil)
		if err != nil {
			return nil, err
		}
		items = append(items, resp.Payload.Items...)
		if !resp.Payload.HasMore || resp.Payload.NextCursor == "" {
			break
		}
		next := resp.Payload.NextCursor
		cursor = &next
	}

	return items, nil
}

func getOverridesFromSandboxes(sandboxes []*models.Sandbox) ([]*sandboxWithForward, error) {
	overrides := make([]*sandboxWithForward, 0)
	for _, sandbox := range sandboxes {
		forwards := builder.GetAvailableOverrideMiddlewares(sandbox)
		if len(forwards) == 0 {
			continue
		}

		overrides = append(overrides, &sandboxWithForward{
			Sandbox:  sandbox.Name,
			Forwards: forwards,
		})
	}
	return overrides, nil
}
