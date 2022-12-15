package routegroup

import (
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/docker/go-units"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/models"
)

type routegroupRow struct {
	Name       string `sdtab:"NAME"`
	RoutingKey string `sdtab:"ROUTING KEY"`
	Cluster    string `sdtab:"CLUSTER"`
	Created    string `sdtab:"CREATED"`
	Status     string `sdtab:"STATUS"`
}

func printRouteGroupTable(out io.Writer, rgs []*models.RouteGroup) error {
	t := sdtab.New[routegroupRow](out)
	t.AddHeader()
	for _, rg := range rgs {
		t.AddRow(routegroupRow{
			Name:       rg.Name,
			RoutingKey: rg.RoutingKey,
			Cluster:    *rg.Spec.Cluster,
			Created:    rg.CreatedAt,
			Status:     readiness(rg.Status),
		})
	}
	return t.Flush()
}

func printRouteGroupDetails(cfg *config.Sandbox, out io.Writer, rg *models.RouteGroup) error {
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)

	fmt.Fprintf(tw, "Name:\t%s\n", rg.Name)
	fmt.Fprintf(tw, "Routing Key:\t%s\n", rg.RoutingKey)
	fmt.Fprintf(tw, "Cluster:\t%s\n", *rg.Spec.Cluster)
	fmt.Fprintf(tw, "Created:\t%s\n", formatTimestamp(rg.CreatedAt))
	fmt.Fprintf(tw, "Dashboard page:\t%s\n", cfg.RouteGroupDashboardURL(rg.Name))
	fmt.Fprintf(tw, "Status:\t%s (%s: %s)\n", readiness(rg.Status), rg.Status.Reason, rg.Status.Message)

	if err := tw.Flush(); err != nil {
		return err
	}

	if len(rg.Endpoints) > 0 {
		fmt.Fprintln(out)
		if err := printEndpointTable(out, rg.Endpoints); err != nil {
			return err
		}
	}

	return nil
}

func readiness(status *models.SandboxReadiness) string {
	if status.Ready {
		return "Ready"
	}
	return "Not Ready"
}

func formatTimestamp(in string) string {
	t, err := time.Parse(time.RFC3339, in)
	if err != nil {
		return in
	}
	elapsed := units.HumanDuration(time.Since(t))
	local := t.Local().Format(time.RFC1123)

	return fmt.Sprintf("%s (%s ago)", local, elapsed)
}

type endpointRow struct {
	Name   string `sdtab:"SANDBOX ENDPOINT"`
	Target string `sdtab:"TARGET"`
	URL    string `sdtab:"URL"`
}

func printEndpointTable(out io.Writer, endpoints []*models.RouteGroupEndpoint) error {
	t := sdtab.New[endpointRow](out)
	t.AddHeader()
	for _, ep := range endpoints {
		t.AddRow(endpointRow{
			Name: ep.Name,
			Type: ep.Target,
			URL:  ep.URL,
		})
	}
	return t.Flush()
}
