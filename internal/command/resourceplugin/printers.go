package resourceplugin

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
)

// displayVersion returns the version component to render in a table cell.
// A bare wire name (no `@version` suffix) means the plugin was published at
// the default version; show that explicitly as 0.0.0 rather than leaving the
// column blank, so a reader can't confuse "default" with "missing data".
func displayVersion(wireName string) string {
	_, version := splitNameVersion(wireName)
	if version == "" {
		return "0.0.0"
	}
	return version
}

// resourcePluginRow is the row shape for `rp list`. The NAME and VERSION
// columns are split (the wire form `name[@version]` is condensed for input
// but expanded for output) so a script can grab the bare name and the version
// independently.
type resourcePluginRow struct {
	Name    string `sdtab:"NAME"`
	Version string `sdtab:"VERSION"`
	Created string `sdtab:"CREATED"`
	Status  string `sdtab:"STATUS"`
}

func printResourcePluginTable(out io.Writer, rps []*models.ResourcePlugin) error {
	t := sdtab.New[resourcePluginRow](out)
	t.AddHeader()
	for _, rp := range rps {
		bareName, _ := splitNameVersion(rp.Name)
		t.AddRow(resourcePluginRow{
			Name:    bareName,
			Version: displayVersion(rp.Name),
			Created: utils.TimeAgo(rp.CreatedAt),
			Status:  status(rp.Status),
		})
	}
	return t.Flush()
}

// resourcePluginVersionRow is the row shape for `rp versions NAME`. The NAME
// column is omitted because the caller supplied the name as an argument;
// repeating it on every row was noise.
type resourcePluginVersionRow struct {
	Version string `sdtab:"VERSION"`
	Created string `sdtab:"CREATED"`
	Status  string `sdtab:"STATUS"`
}

func printResourcePluginVersionsTable(out io.Writer, rps []*models.ResourcePlugin) error {
	t := sdtab.New[resourcePluginVersionRow](out)
	t.AddHeader()
	for _, rp := range rps {
		t.AddRow(resourcePluginVersionRow{
			Version: displayVersion(rp.Name),
			Created: utils.TimeAgo(rp.CreatedAt),
			Status:  status(rp.Status),
		})
	}
	return t.Flush()
}

func printResourcePluginDetails(cfg *config.ResourcePlugin, out io.Writer, rp *models.ResourcePlugin) error {
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)

	fmt.Fprintf(tw, "Name:\t%s\n", rp.Name)
	fmt.Fprintf(tw, "Created:\t%s\n", utils.FormatTimestamp(rp.CreatedAt))
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
