package resourceplugin

import (
	"errors"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	resourceplugins "github.com/signadot/go-sdk/client/resource_plugins"
	"github.com/spf13/cobra"
)

func newDelete(resourcePlugin *config.ResourcePlugin) *cobra.Command {
	cfg := &config.ResourcePluginDelete{ResourcePlugin: resourcePlugin}

	cmd := &cobra.Command{
		Use:   "delete { NAME | -f FILENAME [ --set var1=val1 --set var2=val2 ... ] }",
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
	// Get the name either from a file or from the command line.
	var name string
	if cfg.Filename == "" {
		if len(args) == 0 {
			return errors.New("must specify filename (-f) or resource plugin name")
		}
		if len(cfg.TemplateVals) != 0 {
			return errors.New("must specify filename (-f) to use --set")
		}
		name = args[0]
	} else {
		if len(args) != 0 {
			return errors.New("must not provide args when filename (-f) specified")
		}
		rp, err := loadResourcePlugin(cfg.Filename, cfg.TemplateVals, true /* forDelete */)
		if err != nil {
			return err
		}
		name = rp.Name
	}

	if name == "" {
		return errors.New("resource plugin name is required")
	}

	// Delete the resource plugin.
	params := resourceplugins.NewDeleteResourcePluginParams().
		WithOrgName(cfg.Org).
		WithPluginName(name)
	_, err := cfg.Client.ResourcePlugins.DeleteResourcePlugin(params, nil)
	if err != nil {
		return err
	}

	fmt.Fprintf(log, "Deleted resource plugin %q.\n\n", name)
	return nil
}
