package sandbox

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/docker/go-units"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/models"
)

type sandboxRow struct {
	Name        string `sdtab:"NAME"`
	Description string `sdtab:"DESCRIPTION,trunc"`
	Cluster     string `sdtab:"CLUSTER"`
	Created     string `sdtab:"CREATED"`
	Status      string `sdtab:"STATUS"`
}

func printSandboxTable(out io.Writer, sbs []*models.Sandbox) error {
	t := sdtab.New[sandboxRow](out)
	t.AddHeader()
	for _, sb := range sbs {
		t.AddRow(sandboxRow{
			Name:        sb.Name,
			Description: sb.Spec.Description,
			Cluster:     *sb.Spec.Cluster,
			Created:     sb.CreatedAt,
			Status:      readiness(sb.Status),
		})
	}
	return t.Flush()
}

func printSandboxDetails(cfg *config.Sandbox, out io.Writer, sb *models.Sandbox) error {
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)

	fmt.Fprintf(tw, "ID:\t%s\n", sb.RoutingKey)
	fmt.Fprintf(tw, "Name:\t%s\n", sb.Name)
	fmt.Fprintf(tw, "Description:\t%s\n", sb.Spec.Description)
	fmt.Fprintf(tw, "Cluster:\t%s\n", *sb.Spec.Cluster)
	fmt.Fprintf(tw, "Created:\t%s\n", formatTimestamp(sb.CreatedAt))
	fmt.Fprintf(tw, "Dashboard page:\t%s\n", cfg.SandboxDashboardURL(sb.RoutingKey))
	fmt.Fprintf(tw, "Status:\t%s (%s: %s)\n", readiness(sb.Status), sb.Status.Reason, sb.Status.Message)

	if err := tw.Flush(); err != nil {
		return err
	}

	if len(sb.PreviewEndpoints) > 0 {
		fmt.Fprintln(out)
		if err := printEndpointTable(out, sb.PreviewEndpoints); err != nil {
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
	Name string `sdtab:"PREVIEW ENDPOINT"`
	Type string `sdtab:"TYPE"`
	URL  string `sdtab:"URL"`
}

func printEndpointTable(out io.Writer, endpoints []*models.SandboxPreviewEndpoint) error {
	t := sdtab.New[endpointRow](out)
	t.AddHeader()
	for _, ep := range endpoints {
		t.AddRow(endpointRow{
			Name: ep.Name,
			Type: ep.RouteType,
			URL:  ep.PreviewURL,
		})
	}
	return t.Flush()
}
