package routegroup

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	routegroup "github.com/signadot/go-sdk/client/route_groups"
	"github.com/spf13/cobra"
)

func newList(routegroup *config.RouteGroup) *cobra.Command {
	cfg := &config.RouteGroupList{RouteGroup: routegroup}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List routegroups",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return list(cfg, cmd.OutOrStdout())
		},
	}

	return cmd
}

func list(cfg *config.RouteGroupList, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	resp, err := cfg.Client.RouteGroups.ListRouteGroups(routegroups.NewListRouteGroupsParams().WithOrgName(cfg.Org), nil)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printRouteGroupTable(out, resp.Payload)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
