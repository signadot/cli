package jobs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/signadot/cli/internal/command/logs"
	"github.com/signadot/cli/internal/poll"

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
	if cfg.Timeout > 0 {
		// bound log streaming too, not only the polling below
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	delayTime := 2 * time.Second

	retry := poll.
		NewPoll().
		WithDelay(delayTime).
		WithTimeout(cfg.Timeout)

	ls := &jobLogStreamer{cfg: cfg.API, outW: outW, errW: errW, jobName: jobName}
	queuedLine := false // whether the last line printed is the "Queued" one
	sawRunning := false
	canceled := false

	// finish handles a terminal phase, returning true if phase is terminal.
	finish := func(ctx context.Context, j *models.Job) bool {
		phase := j.Status.Attempts[0].Phase
		switch phase {
		case "succeeded", "failed", "canceled":
		default:
			return false
		}
		// print any logs not yet shown: the job may have gone from queued
		// to terminal between polls, and streams may have stopped early.
		if phase != "canceled" || sawRunning {
			ls.stream(ctx)
		}
		switch phase {
		case "failed":
			handleFailedJobPhase(errW, j)
		case "canceled":
			fmt.Fprintf(outW, "The job execution was canceled\n")
			canceled = true
		}
		return true
	}

	err := retry.Until(ctx, func(ctx context.Context) bool {
		j, err := getJob(cfg.Job, jobName)
		if err != nil {
			fmt.Fprintf(errW, "Error getting job: %s", err.Error())
			queuedLine = false

			// We want to keep retrying if the timeout has not been exceeded
			return false
		}

		// Increases the time, so if the queue is empty will be likely to start
		// seeing the logs right away without any bigger delay
		if delayTime < MaxTimeBetweenRefresh {
			delayTime = (1 * time.Second) + delayTime
			retry.WithDelay(delayTime)
		}

		if finish(ctx, j) {
			return true
		}

		switch j.Status.Attempts[0].Phase {
		case "queued":
			if queuedLine {
				clearLastLine(outW)
			}
			fmt.Fprintf(outW, "Queued on Job Runner Group %s\n", j.Spec.RunnerGroup)
			queuedLine = true

		case "running":
			if queuedLine {
				clearLastLine(outW)
				queuedLine = false
			}
			sawRunning = true
			ls.stream(ctx)

			if j, err = getJob(cfg.Job, jobName); err == nil {
				return finish(ctx, j)
			}
		}
		return false
	})
	if err == nil && canceled {
		err = fmt.Errorf("job %q canceled", jobName)
	}

	return err
}

// jobLogStreamer streams a job's stdout and stderr, resuming each from where
// it last stopped.
type jobLogStreamer struct {
	cfg        *config.API
	outW, errW io.Writer
	jobName    string
	outCursor  string
	errCursor  string
}

// stream streams stdout and stderr concurrently until both end (the server
// ends each independently once the job has finished) or ctx is done. Errors
// are reported to errW.
func (ls *jobLogStreamer) stream(ctx context.Context) {
	var wg sync.WaitGroup
	var outErr, errErr error
	wg.Add(2)
	go func() {
		defer wg.Done()
		outErr = ls.streamOne(ctx, ls.outW, utils.LogTypeStdout, &ls.outCursor)
	}()
	go func() {
		defer wg.Done()
		errErr = ls.streamOne(ctx, ls.errW, utils.LogTypeStderr, &ls.errCursor)
	}()
	wg.Wait()
	if err := errors.Join(outErr, errErr); err != nil {
		fmt.Fprintf(ls.errW, "Error getting logs: %s\n", err.Error())
	}
}

func (ls *jobLogStreamer) streamOne(ctx context.Context, w io.Writer, logType string, cursor *string) error {
	// ShowLogs (re)initializes the API config it is given, so give each
	// concurrent stream its own copy.
	apiCfg := *ls.cfg
	c, err := logs.ShowLogs(ctx, &apiCfg, w, ls.jobName, logType, *cursor, 0)
	if c != "" {
		// keep progress even on error, so a retry doesn't repeat lines
		*cursor = c
	}
	if ctx.Err() != nil {
		return nil // canceled or timed out; reported by the caller
	}
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
