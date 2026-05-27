package resourceplugin

import (
	"errors"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	resourceplugins "github.com/signadot/go-sdk/client/resource_plugins"
	"github.com/signadot/go-sdk/transport"
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
		// The go-sdk's transport middleware (FixAPIErrors) intercepts
		// 4xx/5xx responses before the per-endpoint typed reader runs,
		// so we match on *transport.APIError + Code rather than on
		// *ApplyResourcePluginConflict (which is never in the chain).
		var apiErr *transport.APIError
		if errors.As(err, &apiErr) && apiErr.Code == 409 {
			if version == "" {
				return fmt.Errorf("resource plugin %q default version (0.0.0) already exists; versions are immutable — add an @<semver> suffix to publish a new revision (e.g. %s@0.0.1)", bareName, bareName)
			}
			return fmt.Errorf("resource plugin %q version %q already exists; versions are immutable — bump the @<semver> suffix on name: to publish a new revision", bareName, version)
		}
		return err
	}
	// Always echo the explicit version on the success line, even for
	// bare-name authors (renders as "@0.0.0"). The list / get / table
	// rendering still bares the default version for backward compat;
	// this one message is explicit on purpose, as a teaching moment for
	// first-time-versioning authors.
	fmt.Fprintf(log, "Created resource plugin %s\n\n", explicitNameRef(bareName, version))
	return nil
}

// explicitNameRef renders name@version, defaulting an empty version to
// "0.0.0" so the output names the version that was actually published.
func explicitNameRef(name, version string) string {
	if version == "" {
		version = "0.0.0"
	}
	return name + "@" + version
}
