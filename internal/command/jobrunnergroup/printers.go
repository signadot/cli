package jobrunnergroup

import (
	"fmt"
	"github.com/signadot/cli/internal/utils"
	"github.com/xeonx/timeago"
	"io"
	"time"

	"text/tabwriter"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/models"
)

type runnerGroupRow struct {
	Name    string `sdtab:"NAME"`
	Cluster string `sdtab:"CLUSTER"`
	Created string `sdtab:"CREATED"`
	Status  string `sdtab:"STATUS"`
}

func printRunnerGroupTable(cfg *config.JobRunnerGroupList, out io.Writer, rgs []*models.RunnergroupsRunnerGroup) error {
	t := sdtab.New[runnerGroupRow](out)
	t.AddHeader()
	for _, rg := range rgs {
		createdAt, err := time.Parse(time.RFC3339, rg.CreatedAt)
		if err != nil {
			return err
		}

		t.AddRow(runnerGroupRow{
			Name:    rg.Name,
			Cluster: rg.Spec.Cluster,
			Created: timeago.NoMax(timeago.English).Format(createdAt),
			Status:  readiness(rg.Status),
		})
	}
	return t.Flush()
}

func printRunnerGroupDetails(cfg *config.JobRunnerGroup, out io.Writer, rg *models.RunnergroupsRunnerGroup) error {
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)

	fmt.Fprintf(tw, "Name:\t%s\n", rg.Name)
	fmt.Fprintf(tw, "Created:\t%s\n", utils.FormatTimestamp(rg.CreatedAt))
	fmt.Fprintf(tw, "Status:\t%s\n", readiness(rg.Status))
	fmt.Fprintf(tw, "Dashboard page:\t%s\n", cfg.RunnerGroupDashboardUrl(rg.Name))

	if err := tw.Flush(); err != nil {
		return err
	}

	return nil
}

func readiness(status *models.RunnergroupsStatus) string {
	return fmt.Sprintf("%d/%d runners ready", status.Pods.Ready+status.Pods.Idle, status.Pods.Ready+status.Pods.NotReady+status.Pods.Idle)
}
