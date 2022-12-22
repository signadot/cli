package routegroup

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	routegroups "github.com/signadot/go-sdk/client/route_groups"
	"github.com/spf13/cobra"
)

func newGet(routegroup *config.RouteGroup) *cobra.Command {
	cfg := &config.RouteGroupGet{RouteGroup: routegroup}

	cmd := &cobra.Command{
		Use:   "get NAME",
		Short: "Get routegroup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return get(cfg, cmd.OutOrStdout(), args[0])
		},
	}

	return cmd
}

func get(cfg *config.RouteGroupGet, out io.Writer, name string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	params := routegroups.NewGetRoutegroupParams().WithOrgName(cfg.Org).WithRoutegroupName(name)
	resp, err := cfg.Client.RouteGroups.GetRoutegroup(params, nil)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printRouteGroupDetails(cfg.RouteGroup, out, resp.Payload)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
