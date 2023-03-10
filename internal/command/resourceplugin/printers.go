package resourceplugin

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

type resourcePluginRow struct {
	Name    string `sdtab:"NAME"`
	Created string `sdtab:"CREATED"`
	Status  string `sdtab:"STATUS"`
}

func printResourcePluginTable(out io.Writer, rps []*models.ResourcePlugin) error {
	t := sdtab.New[resourcePluginRow](out)
	t.AddHeader()
	for _, rp := range rps {
		t.AddRow(resourcePluginRow{
			Name:    rp.Name,
			Created: rp.CreatedAt,
			Status:  status(rp.Status),
		})
	}
	return t.Flush()
}

func printResourcePluginDetails(cfg *config.ResourcePlugin, out io.Writer, rp *models.ResourcePlugin) error {
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)

	fmt.Fprintf(tw, "Name:\t%s\n", rp.Name)
	fmt.Fprintf(tw, "Created:\t%s\n", formatTimestamp(rp.CreatedAt))
	fmt.Fprintf(tw, "Status:\t%s\n", status(rp.Status))

	if err := tw.Flush(); err != nil {
		return err
	}

	if len(rp.Status.Resources) > 0 {
		fmt.Fprintln(out)
		if err := printResourcesTable(out, rp.Status.Resources); err != nil {
			return err
		}
	}
	return nil
}

func status(status *models.ResourcepluginStatus) string {
	return fmt.Sprintf("%d resources", len(status.Resources))
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

type resourceRow struct {
	Name    string `sdtab:"RESOURCE NAME"`
	Sandbox string `sdtab:"SANDBOX"`
	Cluster string `sdtab:"CLUSTER"`
}

func printResourcesTable(out io.Writer, resources []*models.ResourceInfo) error {
	t := sdtab.New[resourceRow](out)
	t.AddHeader()
	for _, resource := range resources {
		t.AddRow(resourceRow{
			Name:    resource.Name,
			Sandbox: resource.Sandbox,
			Cluster: resource.Cluster,
		})
	}
	return t.Flush()
}
