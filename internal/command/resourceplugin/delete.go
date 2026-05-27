package resourceplugin

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/signadot/cli/internal/config"
	resourceplugins "github.com/signadot/go-sdk/client/resource_plugins"
	"github.com/signadot/go-sdk/transport"
	"github.com/spf13/cobra"
)

func newDelete(resourcePlugin *config.ResourcePlugin) *cobra.Command {
	cfg := &config.ResourcePluginDelete{ResourcePlugin: resourcePlugin}

	cmd := &cobra.Command{
		Use:   "delete { NAME[@VERSION] | -f FILENAME [ --set var1=val1 --set var2=val2 ... ] }",
		Short: "Delete resource plugin",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return rpDelete(cfg, cmd.ErrOrStderr(), args)
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func rpDelete(cfg *config.ResourcePluginDelete, log io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	// Get the name (and optional version) either from a file or from the command line.
	var name, version string
	if cfg.Filename == "" {
		if len(args) == 0 {
			return errors.New("must specify filename (-f) or resource plugin name")
		}
		if len(cfg.TemplateVals) != 0 {
			return errors.New("must specify filename (-f) to use --set")
		}
		name, version = splitNameVersion(args[0])
	} else {
		if len(args) != 0 {
			return errors.New("must not provide args when filename (-f) specified")
		}
		rp, err := loadResourcePlugin(cfg.Filename, cfg.TemplateVals, true /* forDelete */)
		if err != nil {
			return err
		}
		// rp.Name carries the combined wire form ("bareName[@semver]").
		name, version = splitNameVersion(rp.Name)
	}

	if name == "" {
		return errors.New("resource plugin name is required")
	}

	// For a bare-form delete, resolve the latest version up front so we
	// can both (a) tell the user exactly which version got removed and
	// (b) warn them if other versions remain. Without this, "delete foo"
	// on a plugin that has been bumped past 0.0.0 silently removes only
	// the newest version while the older ones stay — a footgun for users
	// who assumed bare means "the plugin", not "the latest version of
	// the plugin". A small race remains (someone publishes a newer
	// version between this List and the Delete below); we still delete
	// the version we resolved here, which matches the user's mental
	// model: "delete whatever was latest when I asked".
	remainingAfterBareDelete := 0
	if version == "" {
		listParams := resourceplugins.NewListResourcePluginVersionsParams().
			WithOrgName(cfg.Org).WithPluginName(name)
		resp, err := cfg.Client.ResourcePlugins.ListResourcePluginVersions(listParams, nil)
		if err != nil {
			return fmt.Errorf("resolving latest version of %q to delete: %w", name, err)
		}
		if len(resp.Payload) == 0 {
			return fmt.Errorf("resource plugin %q has no published versions to delete", name)
		}
		// ListResourcePluginVersions sorts highest-semver first.
		_, version = splitNameVersion(resp.Payload[0].Name)
		if version == "" {
			// Default-version plugins render bare on the wire; we
			// still need the explicit "0.0.0" for the delete query
			// parameter and the success message.
			version = "0.0.0"
		}
		remainingAfterBareDelete = len(resp.Payload) - 1
	}

	// Delete the resource plugin.
	params := resourceplugins.NewDeleteResourcePluginParams().
		WithOrgName(cfg.Org).
		WithPluginName(name).
		WithVersion(&version)
	_, err := cfg.Client.ResourcePlugins.DeleteResourcePlugin(params, nil)
	if err != nil {
		return enrichDeleteError(cfg, name, version, err)
	}

	fmt.Fprintf(log, "Deleted resource plugin %s.\n", formatNameRef(name, version))
	if remainingAfterBareDelete > 0 {
		fmt.Fprintf(log, "%d other version(s) remain (use 'signadot resourceplugin versions %s' to list).\n",
			remainingAfterBareDelete, name)
	}
	fmt.Fprintln(log)
	return nil
}

// enrichDeleteError converts the generic "plugin is currently in use" 400
// from the server into a message that names the holding sandbox(es), by
// fetching the plugin's Status.Resources. Falls back to the original error
// if the enrichment can't be done.
func enrichDeleteError(cfg *config.ResourcePluginDelete, name, version string, err error) error {
	var apiErr *transport.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != 400 || !strings.Contains(apiErr.Error(), "in use") {
		return err
	}
	getParams := resourceplugins.NewGetResourcePluginParams().
		WithOrgName(cfg.Org).
		WithPluginName(name).
		WithVersion(&version)
	resp, getErr := cfg.Client.ResourcePlugins.GetResourcePlugin(getParams, nil)
	if getErr != nil || resp.Payload == nil || resp.Payload.Status == nil || len(resp.Payload.Status.Resources) == 0 {
		return err
	}
	seen := map[string]bool{}
	sandboxes := []string{}
	for _, r := range resp.Payload.Status.Resources {
		if seen[r.Sandbox] {
			continue
		}
		seen[r.Sandbox] = true
		sandboxes = append(sandboxes, r.Sandbox)
	}
	return fmt.Errorf("resource plugin %s is still referenced by sandbox(es): %s",
		formatNameRef(name, version), strings.Join(sandboxes, ", "))
}
