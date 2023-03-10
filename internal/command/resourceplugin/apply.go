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

	params := resourceplugins.NewApplyResourcePluginParams().
		WithOrgName(cfg.Org).WithPluginName(req.Name).WithData(req)
	_, err = cfg.Client.ResourcePlugins.ApplyResourcePlugin(params, nil)
	if err != nil {
		return err
	}
	fmt.Fprintf(log, "Created resource plugin with name %q\n\n", req.Name)
	return nil
}
