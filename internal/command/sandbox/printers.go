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
}

func printSandboxTable(out io.Writer, sbs []*models.SandboxInfo) error {
	t := sdtab.New[sandboxRow](out)
	t.AddHeader()
	for _, sbinfo := range sbs {
		t.AddRow(sandboxRow{
			Name:        sbinfo.Name,
			Description: sbinfo.Description,
			Cluster:     sbinfo.ClusterName,
			Created:     sbinfo.CreatedAt,
		})
	}
	return t.Flush()
}

func printSandboxDetails(cfg *config.Sandbox, out io.Writer, sb *models.SandboxInfo) error {
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)

	fmt.Fprintf(tw, "ID:\t%s\n", sb.ID)
	fmt.Fprintf(tw, "Name:\t%s\n", sb.Name)
	fmt.Fprintf(tw, "Description:\t%s\n", sb.Description)
	fmt.Fprintf(tw, "Cluster:\t%s\n", sb.ClusterName)
	fmt.Fprintf(tw, "Created:\t%s\n", formatTimestamp(sb.CreatedAt))
	fmt.Fprintf(tw, "Dashboard page:\t%s\n", cfg.SandboxDashboardURL(sb.ID))

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
	Desc string `sdtab:"PREVIEW ENDPOINT"`
	URL  string `sdtab:"URL"`
}

func printEndpointTable(out io.Writer, endpoints []*models.PreviewEndpoint) error {
	t := sdtab.New[endpointRow](out)
	t.AddHeader()
	for _, ep := range endpoints {
		desc := ep.Name
		if ep.ForkOf != nil {
			desc = fmt.Sprintf("Fork of %s/%s", *ep.ForkOf.Namespace, *ep.ForkOf.Name)
		}
		t.AddRow(endpointRow{
			Desc: desc,
			URL:  ep.PreviewURL,
		})
	}
	return t.Flush()
}
