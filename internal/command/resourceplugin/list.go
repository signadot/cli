package resourceplugin

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	resourceplugins "github.com/signadot/go-sdk/client/resource_plugins"
	"github.com/spf13/cobra"
)

func newList(responsePlugin *config.ResourcePlugin) *cobra.Command {
	cfg := &config.ResourcePluginList{ResourcePlugin: responsePlugin}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List resource plugins",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return list(cfg, cmd.OutOrStdout())
		},
	}

	return cmd
}

func list(cfg *config.ResourcePluginList, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	resp, err := cfg.Client.ResourcePlugins.ListResourcePlugins(resourceplugins.NewListResourcePluginsParams().WithOrgName(cfg.Org), nil)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printResourcePluginTable(out, resp.Payload)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
