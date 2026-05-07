package planrunnergroup

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
)

type planRunnerGroupRow struct {
	Name    string `sdtab:"NAME"`
	Cluster string `sdtab:"CLUSTER"`
	Created string `sdtab:"CREATED"`
	Status  string `sdtab:"STATUS"`
}

func printPlanRunnerGroupTable(out io.Writer, prgs []*models.PlanRunnerGroup) error {
	t := sdtab.New[planRunnerGroupRow](out)
	t.AddHeader()
	for _, prg := range prgs {
		t.AddRow(planRunnerGroupRow{
			Name:    prg.Name,
			Cluster: prg.Spec.Cluster,
			Created: utils.TimeAgo(prg.CreatedAt),
			Status:  readiness(prg),
		})
	}
	return t.Flush()
}

func printPlanRunnerGroupDetails(out io.Writer, prg *models.PlanRunnerGroup) error {
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)

	fmt.Fprintf(tw, "Name:\t%s\n", prg.Name)
	fmt.Fprintf(tw, "Cluster:\t%s\n", prg.Spec.Cluster)
	fmt.Fprintf(tw, "Created:\t%s\n", utils.FormatTimestamp(prg.CreatedAt))
	fmt.Fprintf(tw, "Status:\t%s\n", readiness(prg))

	return tw.Flush()
}

func readiness(prg *models.PlanRunnerGroup) string {
	if prg.DeletedAt != "" {
		return "draining"
	}
	if prg.Status == nil || prg.Status.Pods == nil {
		return "-"
	}
	return fmt.Sprintf("%d/%d pods ready",
		prg.Status.Pods.Ready, prg.Status.Pods.Ready+prg.Status.Pods.NotReady)
}
