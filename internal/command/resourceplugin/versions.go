package resourceplugin

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	resourceplugins "github.com/signadot/go-sdk/client/resource_plugins"
	"github.com/spf13/cobra"
)

func newVersions(resourcePlugin *config.ResourcePlugin) *cobra.Command {
	cfg := &config.ResourcePluginVersions{ResourcePlugin: resourcePlugin}

	cmd := &cobra.Command{
		Use:   "versions NAME",
		Short: "List every published version of a resource plugin (highest semver first)",
		Long: `List every published version of a resource plugin, sorted highest-semver first.

The NAME argument is the bare plugin name (no @semver suffix). To fetch a
specific version, use 'signadot resourceplugin get NAME@VERSION'; to fetch
the highest-semver version, use 'signadot resourceplugin get NAME' or
'signadot resourceplugin get NAME@latest'.

Returns an error if no plugin with NAME has ever been published.`,
		Example: `  # Every published version of my-plugin, latest first
  signadot resourceplugin versions my-plugin

  # JSON output for scripts
  signadot resourceplugin versions my-plugin -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return versions(cfg, cmd.OutOrStdout(), args[0])
		},
	}

	return cmd
}

func versions(cfg *config.ResourcePluginVersions, out io.Writer, name string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	params := resourceplugins.NewListResourcePluginVersionsParams().
		WithOrgName(cfg.Org).WithPluginName(name)
	resp, err := cfg.Client.ResourcePlugins.ListResourcePluginVersions(params, nil)
	if err != nil {
		return err
	}
	// Server returns 200 with an empty payload for a name that was never
	// published, instead of 404. A plugin always has at least one version
	// once it exists (publishing creates the (name, version) row), so an
	// empty list reliably means "no such plugin" — surface that as an
	// error rather than exiting 0 with empty rows, matching the behavior
	// of 'resourceplugin get <nonexistent>'.
	if len(resp.Payload) == 0 {
		return fmt.Errorf("resource plugin %q not found", name)
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printResourcePluginVersionsTable(out, resp.Payload)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
