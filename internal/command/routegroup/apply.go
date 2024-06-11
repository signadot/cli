package routegroup

import (
	"errors"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/poll"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/spinner"
	routegroups "github.com/signadot/go-sdk/client/route_groups"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newApply(routegroup *config.RouteGroup) *cobra.Command {
	cfg := &config.RouteGroupApply{RouteGroup: routegroup}

	cmd := &cobra.Command{
		Use:   "apply -f FILENAME [ --set var1=val1 --set var2=val2 ... ]",
		Short: "Create or update a routegroup with variable expansion",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return apply(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func apply(cfg *config.RouteGroupApply, out, log io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	if cfg.Filename == "" {
		return errors.New("must specify routegroup request file with '-f' flag")
	}
	req, err := loadRouteGroup(cfg.Filename, cfg.TemplateVals, false /*forDelete */)
	if err != nil {
		return err
	}

	params := routegroups.NewApplyRoutegroupParams().
		WithOrgName(cfg.Org).WithRoutegroupName(req.Name).WithData(req)
	result, err := cfg.Client.RouteGroups.ApplyRoutegroup(params, nil)
	if err != nil {
		return err
	}
	resp := result.Payload

	fmt.Fprintf(log, "Created routegroup %q (routing key: %s) in cluster %q.\n\n",
		req.Name, resp.RoutingKey, req.Spec.Cluster)

	if cfg.Wait {
		// Wait for the routegroup to be ready.
		// store latest resp for output below
		resp, err = waitForReady(cfg, log, resp)
		if err != nil {
			writeOutput(cfg, out, resp)
			fmt.Fprintf(log, "\nThe routegroup was applied, but it may not be ready yet. To check status, run:\n\n")
			fmt.Fprintf(log, "  signadot routegroup get %v\n\n", req.Name)
			return err
		}
		writeOutput(cfg, out, resp)
		fmt.Fprintf(log, "\nThe routegroup %q was applied and is ready.\n", resp.Name)
		return nil
	}
	return writeOutput(cfg, out, resp)
}

func writeOutput(cfg *config.RouteGroupApply, out io.Writer, resp *models.RouteGroup) error {
	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		// Print info on how to access the routegroup.
		sbURL := cfg.RouteGroup.DashboardURL
		fmt.Fprintf(out, "\nDashboard page: %v\n\n", sbURL)

		if len(resp.Endpoints) > 0 {
			if err := printEndpointTable(out, resp.Endpoints); err != nil {
				return err
			}
		}
		return nil
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}

func waitForReady(cfg *config.RouteGroupApply, out io.Writer, rg *models.RouteGroup) (*models.RouteGroup, error) {
	fmt.Fprintf(out, "Waiting (up to --wait-timeout=%v) for route group to be ready...\n", cfg.WaitTimeout)

	params := routegroups.NewGetRoutegroupParams().WithOrgName(cfg.Org).WithRoutegroupName(rg.Name)

	spin := spinner.Start(out, "Route group status")
	defer spin.Stop()

	retry := poll.
		NewPoll().
		WithTimeout(cfg.WaitTimeout)

	err := retry.Until(func() bool {
		result, err := cfg.Client.RouteGroups.GetRoutegroup(params, nil)
		if err != nil {
			// Keep retrying in case it's a transient error.
			spin.Messagef("error: %v", err)
			return false
		}
		rg = result.Payload
		if !rg.Status.Ready {
			spin.Messagef("Not Ready: %s", rg.Status.Message)
			return false
		}
		spin.StopMessagef("Ready: %s", rg.Status.Message)
		return true
	})
	if err != nil {
		spin.StopFail()
		return rg, err
	}
	return rg, nil
}
