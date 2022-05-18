package print

import (
	"fmt"
	"io"
	"net/url"
	"text/tabwriter"
	"time"

	"github.com/docker/go-units"
	"github.com/signadot/go-sdk/models"
)

type SandboxDetailsConfig interface {
	SandboxDashboardURL(string) *url.URL
}

func SandboxDetails(cfg SandboxDetailsConfig, out io.Writer, sb *models.SandboxInfo) error {
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
		if err := PreviewEndpointTable(out, sb.PreviewEndpoints); err != nil {
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
