package jobs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"sync/atomic"
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

	ls := newJobLogStreamer(cfg.API, outW, errW, jobName)
	queuedLine := false // whether the last line printed is the "Queued" one
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
		// to terminal between polls, and a stream may have been let go before
		// the server ended it. A stream the server ended is not asked again:
		// for a finished job the server ignores the cursor and replays the
		// whole log. A canceled attempt's logs are refused, so there is
		// nothing to drain.
		if phase != "canceled" {
			ls.stream(ctx, true)
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
			ls.stream(ctx, false)

			if j, err = getJob(cfg.Job, jobName); err == nil {
				return finish(ctx, j)
			}
		}
		return false
	})
	if err == nil && canceled {
		err = fmt.Errorf("job %q canceled", jobName)
	}
	if err == nil && ctx.Err() != nil {
		// the --timeout ran out while the last logs were being read
		err = fmt.Errorf("timed out waiting for the logs of job %q; output may be incomplete", jobName)
	}

	return err
}

// logIdleGrace is how long a log stream may stay silent once nothing more is
// expected of it — the other stream has ended, or the job is over — before it
// is let go. The server does not end a stream that never had any output, so
// without it, attaching to a job that writes only to stdout waits on stderr
// until a gateway times the request out.
var logIdleGrace = 3 * time.Second

// logStream is one of a job's two log streams, and how far it has been read.
type logStream struct {
	w       io.Writer
	logType string
	cursor  string
	// ended is whether the server ended the stream, so that everything it
	// held has been shown.
	ended bool
}

// jobLogStreamer streams a job's stdout and stderr, resuming each from where
// it last stopped.
type jobLogStreamer struct {
	cfg      *config.API
	jobName  string
	idle     time.Duration // logIdleGrace, as it was when the streamer was made
	out, err logStream
}

func newJobLogStreamer(cfg *config.API, outW, errW io.Writer, jobName string) *jobLogStreamer {
	return &jobLogStreamer{
		cfg:     cfg,
		jobName: jobName,
		idle:    logIdleGrace,
		out:     logStream{w: outW, logType: utils.LogTypeStdout},
		err:     logStream{w: errW, logType: utils.LogTypeStderr},
	}
}

// stream streams stdout and stderr concurrently until the server ends both, or
// ctx is done. Errors are reported to the stderr writer.
//
// Once one stream ends, the other is let go after the idle grace without
// output. With drain, the job is over: only the streams the server has not
// ended are read, each let go after the idle grace without output from the
// start.
func (ls *jobLogStreamer) stream(ctx context.Context, drain bool) {
	var runs []*logRun
	for _, s := range []*logStream{&ls.out, &ls.err} {
		if drain && s.ended {
			continue
		}
		s.ended = false
		runs = append(runs, newLogRun(ctx, s, ls.idle))
	}
	if len(runs) == 0 {
		return
	}
	done := make(chan *logRun, len(runs))
	for _, r := range runs {
		if drain {
			go r.watch()
		}
		go func(r *logRun) {
			r.err = ls.streamOne(r)
			done <- r
		}(r)
	}
	var errs []error
	for i := range runs {
		r := <-done
		errs = append(errs, r.err)
		if i == 0 && !drain {
			// the first has ended: the other gets the grace from here
			for _, o := range runs {
				if o != r {
					go o.watch()
				}
			}
		}
	}
	for _, r := range runs {
		r.cancel()
	}
	if err := errors.Join(errs...); err != nil {
		fmt.Fprintf(ls.err.w, "Error getting logs: %s\n", err.Error())
	}
}

func (ls *jobLogStreamer) streamOne(r *logRun) error {
	// ShowLogs (re)initializes the API config it is given, so give each
	// concurrent stream its own copy.
	apiCfg := *ls.cfg
	c, err := logs.ShowLogs(r.ctx, &apiCfg, r, ls.jobName, r.s.logType, r.s.cursor, 0)
	if c != "" {
		// keep progress even on error, so a retry doesn't repeat lines
		r.s.cursor = c
	}
	if r.ctx.Err() != nil {
		// let go after the grace, or canceled or timed out (reported by
		// the caller): not ended, and not an error
		return nil
	}
	if err == nil {
		r.s.ended = true
	}
	return err
}

// logRun is one read of a logStream: it writes through to the stream's writer,
// noting when it last did, so that a watch can let it go once it has been
// silent for the idle grace.
type logRun struct {
	s      *logStream
	ctx    context.Context
	cancel context.CancelFunc
	idle   time.Duration
	last   atomic.Int64 // UnixNano of the last write, or of the watch starting
	err    error
}

func newLogRun(ctx context.Context, s *logStream, idle time.Duration) *logRun {
	r := &logRun{s: s, idle: idle}
	r.ctx, r.cancel = context.WithCancel(ctx)
	r.last.Store(time.Now().UnixNano())
	return r
}

func (r *logRun) Write(p []byte) (int, error) {
	r.last.Store(time.Now().UnixNano())
	return r.s.w.Write(p)
}

// watch cancels the run once it has written nothing for the idle grace,
// counted from when the watch starts or the last write after it, and returns
// when the run's context is done.
func (r *logRun) watch() {
	r.last.Store(time.Now().UnixNano())
	tick := time.NewTicker(r.idle / 10)
	defer tick.Stop()
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-tick.C:
			if time.Since(time.Unix(0, r.last.Load())) >= r.idle {
				r.cancel()
				return
			}
		}
	}
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
