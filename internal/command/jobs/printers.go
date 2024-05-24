package jobs

import (
	"fmt"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/client/artifacts"
	"github.com/signadot/go-sdk/models"
	"github.com/xeonx/timeago"
	"io"
	"sort"
	"text/tabwriter"
	"time"
)

type jobRow struct {
	Name        string `sdtab:"NAME"`
	Environment string `sdtab:"ENVIRONMENT"`
	StartedAt   string `sdtab:"STARTED AT"`
	Duration    string `sdtab:"DURATION"`
	Status      string `sdtab:"STATUS"`
}

func printJobTable(cfg *config.JobList, out io.Writer, jobs []*models.JobsJob) error {
	t := sdtab.New[jobRow](out)
	t.AddHeader()

	sort.Slice(jobs, func(i, j int) bool {
		if jobs[i].Status.Attempts[0].Phase == "queued" {
			return true
		}

		if jobs[j].Status.Attempts[0].Phase == "queued" {
			return false
		}

		t1, err1 := time.Parse(time.RFC3339, jobs[i].CreatedAt)
		t2, err2 := time.Parse(time.RFC3339, jobs[j].CreatedAt)
		if err1 != nil || err2 != nil {
			return false
		}

		return t2.Before(t1)
	})
	for _, job := range jobs {
		startedAt, duration := getStartedAndDuration(job)

		environment := ""
		routingContext := job.Spec.RoutingContext
		if routingContext != nil {
			sandboxName := routingContext.Sandbox
			routeGroupName := routingContext.Routegroup
			if len(sandboxName) > 0 {
				environment = fmt.Sprintf("sandbox=%s", sandboxName)
			}

			if len(routeGroupName) > 0 {
				environment += fmt.Sprintf("routegroup=%s", routeGroupName)
			}
		}

		t.AddRow(jobRow{
			Name:        job.Name,
			Environment: environment,
			StartedAt:   startedAt,
			Duration:    duration,
			Status:      job.Status.Phase,
		})
	}
	return t.Flush()
}

func printJobDetails(cfg *config.Job, out io.Writer, job *models.JobsJob) error {
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)

	startedAt, duration := getStartedAndDuration(job)

	fmt.Fprintf(tw, "Job Name:\t%s\n", job.Spec.NamePrefix)
	fmt.Fprintf(tw, "Generated Job Name:\t%s\n", job.Name)
	fmt.Fprintf(tw, "Status:\t%s\n", job.Status.Phase)
	fmt.Fprintln(tw)

	fmt.Fprintf(tw, "Environment:\t%s\n", getJobEnvironment(job))
	fmt.Fprintf(tw, "Started At:\t%s\n", startedAt)
	fmt.Fprintf(tw, "Duration:\t%s\n", duration)

	if err := printArtifacts(cfg, tw, job); err != nil {
		return err
	}

	if err := tw.Flush(); err != nil {
		return err
	}

	return nil
}

func getStartedAndDuration(job *models.JobsJob) (startedAtStr string, durationStr string) {
	var startedAt *time.Time

	if len(job.Status.Attempts) == 0 {
		return "", ""
	}

	attempt := job.Status.Attempts[0]

	if attempt.Phase == "queued" {
		return "", ""
	}

	startedAtRaw := attempt.CreatedAt
	if len(startedAtRaw) != 0 {
		t, err := time.Parse(time.RFC3339, startedAtRaw)
		if err != nil {
			return "", ""
		}

		startedAt = &t
		startedAtStr = timeago.NoMax(timeago.English).Format(t)
	}

	finishedAtRaw := attempt.FinishedAt
	if startedAt != nil && len(finishedAtRaw) != 0 {
		finishedAt, err := time.Parse(time.RFC3339, finishedAtRaw)
		if err != nil {
			return "", ""
		}

		durationTime := finishedAt.Sub(*startedAt)
		durationStr = durationTime.String()
	}

	return startedAtStr, durationStr
}

func getJobEnvironment(job *models.JobsJob) string {
	routingContext := job.Spec.RoutingContext

	if routingContext == nil {
		return "BASELINE"
	}

	if len(routingContext.Sandbox) > 0 {
		return fmt.Sprintf("%s (SANDBOX)", routingContext.Sandbox)
	}

	return fmt.Sprintf("%s (ROUTEGROUP)", routingContext.Routegroup)
}

func getArtifacts(cfg *config.Job, job *models.JobsJob) ([]*models.JobArtifact, error) {
	params := artifacts.NewListJobAttemptArtifactsParams().
		WithOrgName(cfg.Org).
		WithJobAttempt(job.Status.Attempts[0].ID).
		WithJobName(job.Name)

	resp, err := cfg.Client.Artifacts.ListJobAttemptArtifacts(params, nil)
	if err != nil {
		return []*models.JobArtifact{}, nil
	}

	return resp.Payload, nil
}

type jobArtifactRow struct {
	Name string `sdtab:"NAME"`
	Url  string `sdtab:"URL"`
}

func printArtifacts(cfg *config.Job, out io.Writer, job *models.JobsJob) error {
	artifactsList, err := getArtifacts(cfg, job)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "\nArtifacts\n")

	if len(artifactsList) == 0 {
		fmt.Fprintln(out, "No artifacts")
		return nil
	}

	t := sdtab.New[jobArtifactRow](out)
	t.AddHeader()
	for _, artifact := range artifactsList {
		t.AddRow(jobArtifactRow{
			Name: artifact.Path,
			Url:  cfg.ArtifactDownloadUrl(cfg.Org, job.Name, job.Status.Attempts[0].ID, artifact.Path).String(),
		})
	}
	return t.Flush()
}
