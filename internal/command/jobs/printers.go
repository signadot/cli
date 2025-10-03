package jobs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/signadot/cli/internal/command/logs"
	"github.com/signadot/cli/internal/poll"
	"golang.org/x/net/context"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/models"
	"github.com/signadot/go-sdk/utils"
	"github.com/xeonx/timeago"
)

const MaxJobListing = 20
const MaxTimeBetweenRefresh = 10 * time.Second

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
			Status:      string(job.Status.Attempts[0].Phase),
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

	if err := printArtifacts(cfg, tw, job); err != nil {
		return err
	}

	if err := tw.Flush(); err != nil {
		return err
	}

	return nil
}

func waitForJob(ctx context.Context, cfg *config.JobSubmit, outW, errW io.Writer, jobName string) error {
	delayTime := 2 * time.Second

	retry := poll.
		NewPoll().
		WithDelay(delayTime).
		WithTimeout(cfg.Timeout)

	lastOutCursor := ""
	lastErrCursor := ""
	looped := false

	err := retry.Until(func() bool {
		defer func() {
			looped = true
		}()

		j, err := getJob(cfg.Job, jobName)
		if err != nil {
			fmt.Fprintf(errW, "Error getting job: %s", err.Error())

			// We want to keep retrying if the timeout has not been exceeded
			return false
		}

		// Increases the time, so if the queue is empty will be likely to start
		// seeing the logs right away without any bigger delay
		if delayTime < MaxTimeBetweenRefresh {
			delayTime = (1 * time.Second) + delayTime
			retry.WithDelay(delayTime)
		}

		attempt := j.Status.Attempts[0]
		switch attempt.Phase {
		case "succeeded":
			return true

		case "failed":
			handleFailedJobPhase(errW, j)
			return true

		case "queued":
			if looped {
				clearLastLine(outW)
			}

			fmt.Fprintf(outW, "Queued on Job Runner Group %s\n", j.Spec.RunnerGroup)
			return false

		case "running":
			if looped {
				clearLastLine(outW)
			}

			errch := make(chan error)
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			go func() {
				// stream stdout
				cursor, err := logs.ShowLogs(ctx, cfg.API, outW, jobName, utils.LogTypeStdout, lastOutCursor, 0)
				if err == nil {
					lastOutCursor = cursor
				} else if errors.Is(err, context.Canceled) {
					err = nil // ignore context cancelations
				}
				cancel() // this will cause the stderr stream to terminate
				errch <- err
			}()

			go func() {
				// stream stderr
				cursor, err := logs.ShowLogs(ctx, cfg.API, errW, jobName, utils.LogTypeStderr, lastErrCursor, 0)
				if err == nil {
					lastErrCursor = cursor
				} else if errors.Is(err, context.Canceled) {
					err = nil // ignore context cancelations
				}
				cancel() // this will make the stdout stream to terminate
				errch <- err
			}()

			err = errors.Join(<-errch, <-errch) // wait until both streams terminate
			if err != nil {
				fmt.Fprintf(errW, "Error getting logs: %s\n", err.Error())
			}

			if j, err = getJob(cfg.Job, jobName); err == nil {
				switch j.Status.Attempts[0].Phase {
				case "failed":
					handleFailedJobPhase(errW, j)
					return true
				case "succeeded":
					return true
				}
			}
			return false

		case "canceled":
			fmt.Fprintf(outW, "The job execution was canceled\n")
			return true
		}

		return false
	})

	return err
}

func clearLastLine(w io.Writer) {
	fmt.Fprintf(w, "\033[1A\033[K")
}

func handleFailedJobPhase(errW io.Writer, job *models.Job) {
	failedStatus := job.Status.Attempts[0].State.Failed
	if failedStatus.Message != "" {
		fmt.Fprintf(errW, "Error: %s\n", failedStatus.Message)
	}

	exitCode := 1
	if failedStatus.ExitCode != nil && *failedStatus.ExitCode != 0 {
		exitCode = int(*failedStatus.ExitCode)
	}

	os.Exit(exitCode)
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
