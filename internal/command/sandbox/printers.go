package sandbox

import (
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/docker/go-units"
	"github.com/signadot/cli/internal/utils"
	"github.com/xeonx/timeago"

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
		createdAt, err := time.Parse(time.RFC3339, sb.CreatedAt)
		if err != nil {
			return err
		}

		t.AddRow(sandboxRow{
			Name:        sb.Name,
			Description: sb.Spec.Description,
			Cluster:     *sb.Spec.Cluster,
			Created:     timeago.NoMax(timeago.English).Format(createdAt),
			Status:      readiness(sb.Status),
		})
	}
	return t.Flush()
}

func printSandboxDetails(cfg *config.Sandbox, out io.Writer, sb *models.Sandbox) error {
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)

	fmt.Fprintf(tw, "Name:\t%s\n", sb.Name)
	fmt.Fprintf(tw, "Routing Key:\t%s\n", sb.RoutingKey)
	fmt.Fprintf(tw, "Description:\t%s\n", sb.Spec.Description)
	fmt.Fprintf(tw, "Cluster:\t%s\n", *sb.Spec.Cluster)
	fmt.Fprintf(tw, "Created:\t%s\n", utils.FormatTimestamp(sb.CreatedAt))
	fmt.Fprintf(tw, "Updated:\t%s\n", utils.FormatTimestamp(sb.UpdatedAt))
	fmt.Fprintf(tw, "TTL:\t%s\n", formatTTL(sb))
	fmt.Fprintf(tw, "Dashboard page:\t%s\n", cfg.SandboxDashboardURL(sb.Name))
	fmt.Fprintf(tw, "Status:\t%s (%s: %s)\n", readiness(sb.Status), sb.Status.Reason, sb.Status.Message)

	if err := tw.Flush(); err != nil {
		return err
	}

	if len(sb.Endpoints) > 0 {
		fmt.Fprintln(out)
		if err := printEndpointTable(out, sb.Endpoints); err != nil {
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

func formatTTL(sb *models.Sandbox) string {
	ttl := sb.Spec.TTL
	if ttl == nil {
		return "- (forever)"
	}
	var (
		ttlBase time.Time
		err     error
	)
	switch ttl.OffsetFrom {
	case "updatedAt":
		ttlBase, err = time.Parse(time.RFC3339, sb.UpdatedAt)
		if err != nil {
			return fmt.Sprintf("?(e parse-updated-at %q)", sb.UpdatedAt)
		}
	case "createdAt":
		ttlBase, err = time.Parse(time.RFC3339, sb.CreatedAt)
		if err != nil {
			return fmt.Sprintf("?(e parse-created-at %q)", sb.CreatedAt)
		}
	default:
		return fmt.Sprintf("?(bad ttl offset %q)", ttl.OffsetFrom)
	}
	n := len(ttl.Duration)
	count, unit := ttl.Duration[0:n-1], ttl.Duration[n-1:]
	m, err := strconv.ParseInt(count, 10, 32)
	if err != nil {
		return "?(e parse dur)"
	}
	if m < 0 {
		return "?(e negative dur)"
	}
	offset := time.Duration(m)
	switch unit {
	case "m":
		offset *= time.Minute
	case "h":
		offset *= time.Hour
	case "d":
		offset *= 24 * time.Hour
	case "w":
		offset *= 24 * 7 * time.Hour
	}
	eol := ttlBase.Add(offset)
	local := eol.Local().Format(time.RFC1123)
	remaining := eol.Sub(time.Now())
	return fmt.Sprintf("%s (%s)", local, units.HumanDuration(remaining))
}

type endpointRow struct {
	Name string `sdtab:"SANDBOX ENDPOINT"`
	Type string `sdtab:"TYPE"`
	URL  string `sdtab:"URL"`
}

func printEndpointTable(out io.Writer, endpoints []*models.SandboxEndpoint) error {
	t := sdtab.New[endpointRow](out)
	t.AddHeader()
	for _, ep := range endpoints {
		t.AddRow(endpointRow{
			Name: ep.Name,
			Type: ep.RouteType,
			URL:  ep.URL,
		})
	}
	return t.Flush()
}
