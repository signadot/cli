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
		Short: "List resource plugins (highest-semver version of each by default; pass --all-versions to expand)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return list(cfg, cmd.OutOrStdout())
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func list(cfg *config.ResourcePluginList, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	params := resourceplugins.NewListResourcePluginsParams().WithOrgName(cfg.Org)
	if cfg.AllVersions {
		all := "all"
		params = params.WithVersion(&all)
	}
	resp, err := cfg.Client.ResourcePlugins.ListResourcePlugins(params, nil)
	if err != nil {
		return err
	}
	payload := resp.Payload

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		if err := printResourcePluginTable(out, payload); err != nil {
			return err
		}
		// On the default latest-only list, hint at plugins that have
		// more than one version by issuing a cheap second list with
		// ?version=all and counting how many bare names show up more
		// than once. Skipped under --all-versions (every version is
		// already on screen) and on JSON/YAML output.
		if !cfg.AllVersions && len(payload) > 0 {
			if hint, err := multiVersionHint(cfg); err == nil && hint != "" {
				fmt.Fprintln(out, hint)
			}
		}
		return nil
	case config.OutputFormatJSON:
		return print.RawJSON(out, payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}

// multiVersionHint counts how many plugin names have more than one
// published version and returns a one-line footer pointing at
// --all-versions if any do. Silently no-ops on error — the hint is
// supplementary, never load-bearing for the command's exit status.
func multiVersionHint(cfg *config.ResourcePluginList) (string, error) {
	all := "all"
	params := resourceplugins.NewListResourcePluginsParams().
		WithOrgName(cfg.Org).WithVersion(&all)
	resp, err := cfg.Client.ResourcePlugins.ListResourcePlugins(params, nil)
	if err != nil {
		return "", err
	}
	counts := map[string]int{}
	for _, rp := range resp.Payload {
		bare, _ := splitNameVersion(rp.Name)
		counts[bare]++
	}
	multi := 0
	for _, c := range counts {
		if c > 1 {
			multi++
		}
	}
	if multi == 0 {
		return "", nil
	}
	if multi == 1 {
		return "(1 plugin has multiple versions — pass --all-versions to expand, or 'resourceplugin versions NAME')", nil
	}
	return fmt.Sprintf("(%d plugins have multiple versions — pass --all-versions to expand, or 'resourceplugin versions NAME')", multi), nil
}
