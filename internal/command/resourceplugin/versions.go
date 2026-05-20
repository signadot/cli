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
		Short: "List published versions of a resource plugin",
		Args:  cobra.ExactArgs(1),
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
