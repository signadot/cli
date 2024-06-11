package jobs

import (
	"fmt"
	"github.com/signadot/cli/internal/command/logs"
	"github.com/signadot/cli/internal/poll"
	"github.com/signadot/go-sdk/client/jobs"
	"io"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/client/artifacts"
	"github.com/signadot/go-sdk/models"
	"github.com/xeonx/timeago"
)

const MaxJobListing = 20
const TimeBetweenJobRefresh = 10 * time.Second

type jobRow struct {
	Name        string `sdtab:"NAME"`
	Environment string `sdtab:"ENVIRONMENT"`
	CreatedAt   string `sdtab:"CREATED AT"`
	StartedAt   string `sdtab:"STARTED AT"`
	Duration    string `sdtab:"DURATION"`
	Status      string `sdtab:"STATUS"`
}

func printJobTable(cfg *config.JobList, out io.Writer, jobs []*models.Job) error {
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

	counter := 0
	for _, job := range jobs {
		if counter == MaxJobListing {
			break
		}

		switch {
		case cfg.ShowAll:
		case !cfg.ShowAll && job.Status.Phase != "completed" && job.Status.Phase != "canceled":
		default:
			continue
		}

		counter += 1

		createdAt, duration := getAttemptCreatedAtAndDuration(job)

		environment := getJobEnvironment(job)

		t.AddRow(jobRow{
			Name:        job.Name,
			Environment: environment,
			StartedAt:   createdAt,
			Duration:    duration,
			Status:      job.Status.Phase,
			CreatedAt:   getCreatedAt(job),
		})
	}
	return t.Flush()
}

func printJobDetails(cfg *config.JobGet, out io.Writer, job *models.Job) error {
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)

	createdAt, duration := getAttemptCreatedAtAndDuration(job)

	fmt.Fprintf(tw, "Job Name:\t%s\n", job.Name)
	fmt.Fprintf(tw, "Job Runner Group:\t%s\n", job.Spec.RunnerGroup)
	fmt.Fprintf(tw, "Status:\t%s\n", job.Status.Phase)
	fmt.Fprintf(tw, "Environment:\t%s\n", getJobEnvironment(job))
	fmt.Fprintf(tw, "Created At:\t%s\n", getCreatedAt(job))

	if len(createdAt) != 0 {
		fmt.Fprintf(tw, "Started At:\t%s\n", createdAt)
	}

	if len(duration) != 0 {
		fmt.Fprintf(tw, "Duration:\t%s\n", duration)
	}

	fmt.Fprintf(tw, "Dashboard URL:\t%s\n", cfg.JobDashboardUrl(job.Name))

	if err := printArtifacts(cfg, tw, job); err != nil {
		return err
	}

	if err := tw.Flush(); err != nil {
		return err
	}

	return nil
}

func waitForJob(cfg *config.JobSubmit, out io.Writer, jobName string) error {

	fmt.Fprintf(out, "Waiting for job execution\n")

	looped := false

	retry := poll.
		NewPoll().
		WithDelay(TimeBetweenJobRefresh).
		WithTimeout(cfg.WaitTimeout)

	err := retry.UntilWithError(func() (bool, error) {
		j, err := getJob(cfg.Job, jobName)
		if err != nil {
			// We want to keep retrying if the timeout has been exceeded
			return false, nil
		}

		switch j.Status.Phase {
		case "completed":
			os.Exit(int(j.Status.Attempts[0].State.Completed.ExitCode))

			return true, nil
		case "queued":
			if looped {
				fmt.Fprintf(out, "\033[1A\033[K")
			}

			fmt.Fprintf(out, "Queued on Job Runner Group %s\n", j.Spec.RunnerGroup)
		case "running":
			if looped {
				fmt.Fprintf(out, "\033[1A\033[K")
			}

			logsCmd := logs.New(cfg.API)
			logsCmd.SetArgs([]string{"--job=" + jobName, "--stream=" + cfg.Attach})
			if err := logsCmd.Execute(); err != nil {
				return false, err
			}
		case "canceled":
			fmt.Fprintf(out, "Stopping cause job execution was canceled\n")
			return true, nil
		}

		looped = true

		return false, nil
	})

	return err
}

func getJob(cfg *config.Job, jobName string) (*models.Job, error) {
	params := jobs.NewGetJobParams().WithOrgName(cfg.Org).WithJobName(jobName)
	resp, err := cfg.Client.Jobs.GetJob(params, nil)
	if err != nil {
		return nil, err
	}

	return resp.Payload, nil
}

func getCreatedAt(job *models.Job) string {
	createdAt := job.CreatedAt
	if len(createdAt) == 0 {
		return ""
	}

	t, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return ""
	}

	return timeago.NoMax(timeago.English).Format(t)
}

func getAttemptCreatedAtAndDuration(job *models.Job) (createdAtStr string, durationStr string) {
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

func getJobEnvironment(job *models.Job) string {
	routingContext := job.Spec.RoutingContext

	switch {
	case routingContext == nil:
	case len(routingContext.Sandbox) > 0:
		return fmt.Sprintf("sandbox=%s", routingContext.Sandbox)
	case len(routingContext.Routegroup) > 0:
		return fmt.Sprintf("routegroup=%s", routingContext.Routegroup)
	}

	return "baseline"
}

func getArtifacts(cfg *config.JobGet, job *models.Job) ([]*models.JobArtifact, error) {
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
	Path string `sdtab:"PATH"`
	Size string `sdtab:"SIZE"`
}

func printArtifacts(cfg *config.JobGet, out io.Writer, job *models.Job) error {
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
		path := artifact.Path

		if _, ok := excludeFiles[path]; ok {
			continue
		}

		if artifact.Space == "system" {
			path = "@" + path
		}

		t.AddRow(jobArtifactRow{
			Path: path,
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
