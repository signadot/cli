package jobs

import (
	"fmt"
	"io"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/models"
	"github.com/xeonx/timeago"
)

const MaxJobListing = 20

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
		if !cfg.ShowAll && !isJobPhaseToPrintDefault(job.Status.Attempts[0].Phase) {
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
			Status:      job.Status.Attempts[0].Phase,
			CreatedAt:   getCreatedAt(job),
		})
	}
	return t.Flush()
}

func isJobPhaseToPrintDefault(ph string) bool {
	if ph == "failed" {
		return false
	}
	if ph == "succeeded" {
		return false
	}
	if ph == "canceled" {
		return false
	}
	return true
}

func printJobDetails(cfg *config.Job, out io.Writer, job *models.Job, artifacts []*models.JobArtifact) error {
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)

	createdAt, duration := getAttemptCreatedAtAndDuration(job)

	fmt.Fprintf(tw, "Job Name:\t%s\n", job.Name)
	fmt.Fprintf(tw, "Job Runner Group:\t%s\n", job.Spec.RunnerGroup)
	fmt.Fprintf(tw, "Status:\t%s\n", getJobStatus(job))
	if state := job.Status.Attempts[0].State; state != nil {
		switch {
		case state.Queued != nil:
			fmt.Fprintf(tw, "Message:\t%s\n", state.Queued.Message)
		case state.Running != nil:
			fmt.Fprintf(tw, "Runner Pod:\t%s/%s\n", state.Running.PodNamespace, state.Running.PodName)
		case state.Canceled != nil:
			fmt.Fprintf(tw, "Canceled By:\t%s\n", state.Canceled.CanceledBy)
			fmt.Fprintf(tw, "Message:\t%s\n", state.Canceled.Message)
		case state.Failed != nil:
			if state.Failed.ExitCode != nil {
				fmt.Fprintf(tw, "Exit Code:\t%d\n", *state.Failed.ExitCode)
			}
			fmt.Fprintf(tw, "Message:\t%s\n", state.Failed.Message)
		}
	}

	fmt.Fprintf(tw, "Environment:\t%s\n", getJobEnvironment(job))
	fmt.Fprintf(tw, "Created At:\t%s\n", getCreatedAt(job))

	if len(createdAt) != 0 {
		fmt.Fprintf(tw, "Started At:\t%s\n", createdAt)
	}

	if len(duration) != 0 {
		fmt.Fprintf(tw, "Duration:\t%s\n", duration)
	}

	fmt.Fprintf(tw, "Dashboard URL:\t%s\n", cfg.JobDashboardUrl(job.Name))

	if err := printArtifacts(tw, artifacts); err != nil {
		return err
	}

	if err := tw.Flush(); err != nil {
		return err
	}

	return nil
}

func getJobStatus(job *models.Job) string {
	switch job.Status.Attempts[0].Phase {
	case "queued":
		return "Queued"
	case "running":
		return "Running"
	case "failed":
		return "Failed"
	case "succeeded":
		return "Succeeded"
	case "canceled":
		return "Canceled"
	}
	return "Unknown"
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

type jobArtifactRow struct {
	Path string `sdtab:"PATH"`
	Size string `sdtab:"SIZE"`
}

func rangeArtifacts(artifacts []*models.JobArtifact, fn func(path string, size int64)) {
	excludeFiles := map[string]bool{"stderr.index": true, "stdout.index": true}
	for _, artifact := range artifacts {
		path := artifact.Path
		if _, ok := excludeFiles[path]; ok {
			continue
		}

		if artifact.Space == "system" {
			path = "@" + path
		}
		fn(path, artifact.Size)
	}
}

func printArtifacts(out io.Writer, artifacts []*models.JobArtifact) error {
	fmt.Fprintf(out, "\nArtifacts\n")
	if len(artifacts) == 0 {
		fmt.Fprintln(out, "No artifacts")
		return nil
	}

	t := sdtab.New[jobArtifactRow](out)
	t.AddHeader()
	rangeArtifacts(artifacts, func(path string, size int64) {
		t.AddRow(jobArtifactRow{
			Path: path,
			Size: byteCountSI(size),
		})
	})
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

func printRawJob(out io.Writer, printer func(out io.Writer, v any) error,
	job *models.Job, artifacts []*models.JobArtifact) error {
	attempt := job.Status.Attempts[0]

	type rawArtifact struct {
		Path string `json:"path,omitempty"`
		Size int64  `json:"size"`
	}

	type rawJobAttemptStatus struct {
		CreatedAt      string            `json:"createdAt,omitempty"`
		StartedAt      string            `json:"startedAt,omitempty"`
		FinishedAt     string            `json:"finishedAt,omitempty"`
		ExecutionCount int64             `json:"executionCount,omitempty"`
		Phase          string            `json:"phase,omitempty"`
		State          *models.JobsState `json:"state,omitempty"`
		Artifacts      []*rawArtifact    `json:"artifacts,omitempty"`
	}

	type rawJob struct {
		Spec   *models.JobSpec      `json:"spec,omitempty"`
		Status *rawJobAttemptStatus `json:"status,omitempty"`
	}

	displayableArtifacts := make([]*rawArtifact, 0, len(artifacts))
	rangeArtifacts(artifacts, func(path string, size int64) {
		displayableArtifacts = append(displayableArtifacts, &rawArtifact{
			Path: path,
			Size: size,
		})
	})

	displayableJob := &rawJob{
		Spec: job.Spec,
		Status: &rawJobAttemptStatus{
			CreatedAt:      attempt.CreatedAt,
			StartedAt:      attempt.StartedAt,
			FinishedAt:     attempt.FinishedAt,
			ExecutionCount: attempt.ExecutionCount,
			Phase:          attempt.Phase,
			State:          attempt.State,
			Artifacts:      displayableArtifacts,
		},
	}

	return printer(out, displayableJob)
}
