package jobs

import (
	"fmt"
	"io"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/client/artifacts"
	"github.com/signadot/go-sdk/models"
	"github.com/xeonx/timeago"
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
		createdAt, duration := getCreatedAtAndDuration(job)

		environment := ""
		routingContext := job.Spec.RoutingContext
		switch {
		case routingContext == nil:
		case len(routingContext.Sandbox) > 0:
			environment = fmt.Sprintf("sandbox=%s", routingContext.Sandbox)
		case len(routingContext.Routegroup) > 0:
			environment += fmt.Sprintf("routegroup=%s", routingContext.Routegroup)
		}

		t.AddRow(jobRow{
			Name:        job.Name,
			Environment: environment,
			StartedAt:   createdAt,
			Duration:    duration,
			Status:      job.Status.Phase,
		})
	}
	return t.Flush()
}

func printJobDetails(cfg *config.Job, out io.Writer, job *models.JobsJob) error {
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)

	createdAt, duration := getCreatedAtAndDuration(job)

	fmt.Fprintf(tw, "Job Name:\t%s\n", job.Name)
	fmt.Fprintf(tw, "Status:\t%s\n", job.Status.Phase)
	fmt.Fprintf(tw, "Environment:\t%s\n", getJobEnvironment(job))
	fmt.Fprintf(tw, "Started At:\t%s\n", createdAt)
	fmt.Fprintf(tw, "Duration:\t%s\n", duration)
	fmt.Fprintf(tw, "Dashboard URL:\t%s\n", cfg.JobDashboardUrl(job.Name))

	if err := printArtifacts(cfg, tw, job); err != nil {
		return err
	}

	if err := tw.Flush(); err != nil {
		return err
	}

	return nil
}

func getCreatedAtAndDuration(job *models.JobsJob) (createdAtStr string, durationStr string) {
	var createdAt *time.Time

	if len(job.Status.Attempts) == 0 {
		return "", ""
	}

	attempt := job.Status.Attempts[0]

	if attempt.Phase == "queued" {
		return "", ""
	}

	createdAtRaw := attempt.CreatedAt
	if len(createdAtRaw) != 0 {
		t, err := time.Parse(time.RFC3339, createdAtRaw)
		if err != nil {
			return "", ""
		}

		createdAt = &t
		createdAtStr = timeago.NoMax(timeago.English).Format(t)
	}

	finishedAtRaw := attempt.FinishedAt
	if createdAt != nil && len(finishedAtRaw) != 0 {
		finishedAt, err := time.Parse(time.RFC3339, finishedAtRaw)
		if err != nil {
			return "", ""
		}

		durationTime := finishedAt.Sub(*createdAt)
		durationStr = durationTime.String()
	}

	return createdAtStr, durationStr
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
	Size string `sdtab:"SIZE"`
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

	excludeFiles := map[string]bool{"stderr.index": true, "stdout.index": true}
	for _, artifact := range artifactsList {
		name := artifact.Path

		if _, ok := excludeFiles[name]; ok {
			continue
		}

		if artifact.Space == "system" {
			name = "@" + name
		}

		t.AddRow(jobArtifactRow{
			Name: name,
			Size: byteCountSI(artifact.Size),
		})
	}
	return t.Flush()
}

func byteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}
