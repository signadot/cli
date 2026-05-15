package resourceplugin

import (
	"errors"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	resourceplugins "github.com/signadot/go-sdk/client/resource_plugins"
	"github.com/spf13/cobra"
)

func newApply(resourcePlugin *config.ResourcePlugin) *cobra.Command {
	cfg := &config.ResourcePluginApply{ResourcePlugin: resourcePlugin}

	cmd := &cobra.Command{
		Use:   "apply -f FILENAME [ --set var1=val1 --set var2=val2 ... ]",
		Short: "Create or update a resource plugin with variable expansion",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return apply(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func apply(cfg *config.ResourcePluginApply, out, log io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	if cfg.Filename == "" {
		return errors.New("must specify resource plugin request file with '-f' flag")
	}
	req, err := loadResourcePlugin(cfg.Filename, cfg.TemplateVals, false /*forDelete */)
	if err != nil {
		return err
	}

	// req.Name carries the combined wire form ("bareName[@semver]").
	// Split it back for the URL path (which must be the bare name) and
	// for the version description used in user-facing messages.
	bareName, version := splitNameVersion(req.Name)
	params := resourceplugins.NewApplyResourcePluginParams().
		WithOrgName(cfg.Org).WithPluginName(bareName).WithData(req)
	_, err = cfg.Client.ResourcePlugins.ApplyResourcePlugin(params, nil)
	if err != nil {
		var conflict *resourceplugins.ApplyResourcePluginConflict
		if errors.As(err, &conflict) {
			return fmt.Errorf(
				"resource plugin %q %s already exists; versions are immutable — bump the version field to publish a new revision",
				bareName, versionDescription(version))
		}
		return err
	}
	fmt.Fprintf(log, "Created resource plugin %s\n\n", formatNameRef(bareName, version))
	return nil
}

// versionDescription renders the version for user-facing messages, calling out
// the implicit default when the request body omitted version.
func versionDescription(version string) string {
	if version == "" {
		return "default version (0.0.0)"
	}
	return fmt.Sprintf("version %q", version)
}
