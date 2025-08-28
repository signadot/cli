package routegroup

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/xeonx/timeago"

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
	Ready      string `sdtab:"READY SANDBOXES"`
}

func printRouteGroupTable(cfg *config.RouteGroupList, out io.Writer, rgs []*models.RouteGroup) error {
	t := sdtab.New[routegroupRow](out)
	t.AddHeader()
	for _, rg := range rgs {
		sbxStatus, err := getSandboxesStatus(cfg, rg.Status)
		if err != nil {
			return err
		}

		createdAt, err := time.Parse(time.RFC3339, rg.CreatedAt)
		if err != nil {
			return err
		}

		t.AddRow(routegroupRow{
			Name:       rg.Name,
			RoutingKey: rg.RoutingKey,
			Cluster:    rg.Spec.Cluster,
			Created:    timeago.NoMax(timeago.English).Format(createdAt),
			Status:     readiness(rg.Status),
			Ready:      sbxStatus,
		})
	}
	return t.Flush()
}

func printRouteGroupDetails(cfg *config.RouteGroup, out io.Writer, rg *models.RouteGroup) error {
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)

	fmt.Fprintf(tw, "Name:\t%s\n", rg.Name)
	fmt.Fprintf(tw, "Routing Key:\t%s\n", rg.RoutingKey)
	fmt.Fprintf(tw, "Cluster:\t%s\n", rg.Spec.Cluster)
	fmt.Fprintf(tw, "Created:\t%s\n", utils.FormatTimestamp(rg.CreatedAt))
	fmt.Fprintf(tw, "TTL:\t%s\n", formatTTL(rg.Spec, rg.Status.ScheduledDeleteTime))
	fmt.Fprintf(tw, "Dashboard page:\t%s\n", cfg.DashboardURL)
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

func readiness(status *models.RouteGroupStatus) string {
	if status.Ready {
		return "Ready"
	}
	return "Not Ready"
}

func getSandboxesStatus(cfg *config.RouteGroupList, status *models.RouteGroupStatus) (string, error) {
	readyCounter := 0

	matchedSandboxes := status.MatchedSandboxes
	for _, sandboxName := range matchedSandboxes {
		params := sandboxes.NewGetSandboxParams().WithOrgName(cfg.Org).WithSandboxName(sandboxName)
		sandbox, err := cfg.Client.Sandboxes.GetSandbox(params, nil)
		if err != nil {
			return "", err
		}

		if sandbox.Payload.Status.Ready {
			readyCounter += 1
		}
	}

	return fmt.Sprintf("%d/%d", readyCounter, len(matchedSandboxes)), nil
}

func formatTTL(spec *models.RouteGroupSpec, deletionTime string) string {
	if spec.TTL == nil {
		return "- (forever)"
	}

	if deletionTime == "" && spec.TTL.OffsetFrom == "noMatchedSandboxes" {
		return fmt.Sprintf("%s after no matching sanboxes", utils.GetTTLTimeAgoFromBase(time.Now(), spec.TTL.Duration))
	}

	t, err := time.Parse(time.RFC3339, deletionTime)
	if err != nil {
		return deletionTime
	}
	local := t.Local().Format(time.RFC1123)

	return fmt.Sprintf("%s (%s)", local, timeago.NoMax(timeago.English).Format(t))
}

type endpointRow struct {
	Name   string `sdtab:"ROUTEGROUP ENDPOINT"`
	Target string `sdtab:"TARGET"`
	URL    string `sdtab:"URL"`
}

func printEndpointTable(out io.Writer, endpoints []*models.RoutegroupsEndpointURL) error {
	t := sdtab.New[endpointRow](out)
	t.AddHeader()
	for _, ep := range endpoints {
		t.AddRow(endpointRow{
			Name:   ep.Name,
			Target: ep.Target,
			URL:    ep.URL,
		})
	}
	return t.Flush()
}
