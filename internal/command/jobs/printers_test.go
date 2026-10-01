package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/client"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/viper"
)

// fakeJobConfig returns a config for a fake API server reporting job1 in the
// given phase. logs maps a log type (stdout/stderr) to its SSE handler.
func fakeJobConfig(t *testing.T, phase models.JobsPhase, logs map[string]http.HandlerFunc) *config.JobSubmit {
	t.Helper()
	return fakeJobConfigPhases(t, func() models.JobsPhase { return phase }, logs)
}

// fakeJobConfigPhases is fakeJobConfig with the phase asked at each GET of the
// job, so a test can move the job along.
func fakeJobConfigPhases(t *testing.T, phase func() models.JobsPhase, logs map[string]http.HandlerFunc) *config.JobSubmit {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/orgs/org/jobs/job1/attempts/0/logs/stream" {
			if h, ok := logs[r.URL.Query().Get("type")]; ok {
				h(w, r)
				return
			}
		}
		if r.URL.Path != "/api/v2/orgs/org/jobs/job1" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&models.Job{
			Name: "job1",
			Spec: &models.JobSpec{RunnerGroup: "rg"},
			Status: &models.JobsStatus{
				Attempts: []*models.JobsAttempt{{Phase: phase()}},
			},
		})
	}))
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	c := client.NewHTTPClientWithConfig(nil, client.DefaultTransportConfig().
		WithHost(u.Host).WithSchemes([]string{"http"}))
	// used by the log streaming client
	viper.Set("api_key", "key")
	viper.Set("org", "org")
	viper.Set("api_url", srv.URL)
	t.Cleanup(viper.Reset)
	return &config.JobSubmit{
		Job:     &config.Job{API: &config.API{Org: "org", Client: c, APIURL: srv.URL}},
		Timeout: 10 * time.Second,
	}
}

func TestWaitForJobCanceledIsError(t *testing.T) {
	cfg := fakeJobConfig(t, models.JobsPhaseCanceled, nil)
	err := waitForJob(context.Background(), cfg, io.Discard, io.Discard, "job1")
	if err == nil || !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("expected canceled error, got %v", err)
	}
}

func TestWaitForJobSucceeded(t *testing.T) {
	cfg := fakeJobConfig(t, models.JobsPhaseSucceeded, nil)
	if err := waitForJob(context.Background(), cfg, io.Discard, io.Discard, "job1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func sseLogs(delay time.Duration, msg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		w.Header().Set("Content-Type", "text/event-stream")
		d, _ := json.Marshal(map[string]string{"message": msg, "cursor": "c1"})
		fmt.Fprintf(w, "event: message\ndata: %s\n\nevent: signal\ndata: EOF\n\n", d)
	}
}

// A job observed only once already finished must still have its logs shown,
// and stdout ending first must not cut off stderr.
func TestWaitForJobTerminalShowsAllLogs(t *testing.T) {
	cfg := fakeJobConfig(t, models.JobsPhaseSucceeded, map[string]http.HandlerFunc{
		"stdout": sseLogs(0, "out-line\n"),
		"stderr": sseLogs(300*time.Millisecond, "err-line\n"),
	})
	var out, errOut strings.Builder
	if err := waitForJob(context.Background(), cfg, &out, &errOut, "job1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "out-line") {
		t.Errorf("stdout missing log line: %q", out.String())
	}
	if !strings.Contains(errOut.String(), "err-line") {
		t.Errorf("stderr missing log line: %q", errOut.String())
	}
}

// sseEvents writes SSE events: a message per line, then the EOF signal.
func sseEvents(w http.ResponseWriter, lines []string, restart bool) {
	w.Header().Set("Content-Type", "text/event-stream")
	if restart {
		fmt.Fprint(w, "event: signal\ndata: RESTART\n\n")
	}
	for i, l := range lines {
		d, _ := json.Marshal(map[string]string{"message": l + "\n", "cursor": fmt.Sprintf("c%d", i+1)})
		fmt.Fprintf(w, "event: message\ndata: %s\n\n", d)
	}
	fmt.Fprint(w, "event: signal\ndata: EOF\n\n")
}

// finishedLog serves a finished job's log the way the server does: the whole
// log from the start, and asked to resume from a cursor, a RESTART and the
// whole log again.
func finishedLog(lines ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sseEvents(w, lines, r.URL.Query().Get("cursor") != "")
	}
}

// silentLog is a stream the server never ends: one that had no output.
func silentLog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.(http.Flusher).Flush()
	<-r.Context().Done()
}

// runningThen reports running at the first GET of the job and then phase.
func runningThen(phase models.JobsPhase) func() models.JobsPhase {
	var n atomic.Int32
	return func() models.JobsPhase {
		if n.Add(1) == 1 {
			return models.JobsPhaseRunning
		}
		return phase
	}
}

func shortIdleGrace(t *testing.T) {
	t.Helper()
	old := logIdleGrace
	logIdleGrace = 200 * time.Millisecond
	t.Cleanup(func() { logIdleGrace = old })
}

// A job seen running streams its logs to the end; when it is then seen
// finished, the logs are not asked for again — the server would replay them
// whole after a RESTART.
func TestWaitForJobRunningThenDoneShowsLogsOnce(t *testing.T) {
	shortIdleGrace(t)
	cfg := fakeJobConfigPhases(t, runningThen(models.JobsPhaseSucceeded), map[string]http.HandlerFunc{
		"stdout": finishedLog("out 1", "out 2"),
		"stderr": finishedLog("err 1"),
	})
	var out, errOut strings.Builder
	if err := waitForJob(context.Background(), cfg, &out, &errOut, "job1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n := strings.Count(out.String(), "out 1"); n != 1 {
		t.Errorf("stdout shown %d times, want once:\n%s", n, out.String())
	}
	if n := strings.Count(errOut.String(), "err 1"); n != 1 {
		t.Errorf("stderr shown %d times, want once:\n%s", n, errOut.String())
	}
	if strings.Contains(out.String()+errOut.String(), "restarted") {
		t.Errorf("a replay was shown:\n%s\n%s", out.String(), errOut.String())
	}
}

// A stream with no output is never ended by the server; once the other has
// ended, it is let go after the grace rather than held until a gateway times
// the request out.
func TestWaitForJobDoesNotWaitOnASilentStream(t *testing.T) {
	shortIdleGrace(t)
	cfg := fakeJobConfigPhases(t, runningThen(models.JobsPhaseSucceeded), map[string]http.HandlerFunc{
		"stdout": finishedLog("tick 1", "tick 2"),
		"stderr": silentLog,
	})
	var out, errOut strings.Builder
	start := time.Now()
	if err := waitForJob(context.Background(), cfg, &out, &errOut, "job1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("waited %s on a stream with no output", d)
	}
	if n := strings.Count(out.String(), "tick 1"); n != 1 {
		t.Errorf("stdout shown %d times, want once:\n%s", n, out.String())
	}
	if errOut.String() != "" {
		t.Errorf("unexpected stderr: %q", errOut.String())
	}
}

// A canceled attempt's logs are refused by the server, so canceling after the
// job was seen running reports the cancel and nothing about the logs.
func TestWaitForJobCanceledAfterRunningDoesNotAskForLogs(t *testing.T) {
	shortIdleGrace(t)
	phase := runningThen(models.JobsPhaseCanceled)
	var canceled atomic.Bool
	refuseOnceCanceled := func(lines ...string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if canceled.Load() {
				http.Error(w, "canceled job attempt", http.StatusBadRequest)
				return
			}
			sseEvents(w, lines, false)
		}
	}
	cfg := fakeJobConfigPhases(t, func() models.JobsPhase {
		p := phase()
		canceled.Store(p == models.JobsPhaseCanceled)
		return p
	}, map[string]http.HandlerFunc{
		"stdout": refuseOnceCanceled("out 1"),
		"stderr": refuseOnceCanceled("err 1"),
	})
	var out, errOut strings.Builder
	err := waitForJob(context.Background(), cfg, &out, &errOut, "job1")
	if err == nil || !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("expected canceled error, got %v", err)
	}
	if strings.Contains(errOut.String(), "Error getting logs") {
		t.Errorf("the canceled attempt's logs were asked for:\n%s", errOut.String())
	}
}

// --timeout running out while the last logs are read is an error, not a
// success with the output cut short.
func TestWaitForJobTimeoutWhileReadingLogsIsAnError(t *testing.T) {
	shortIdleGrace(t)
	cfg := fakeJobConfig(t, models.JobsPhaseSucceeded, map[string]http.HandlerFunc{
		"stdout": func(w http.ResponseWriter, r *http.Request) {
			// a long log, still arriving when the timeout runs out
			w.Header().Set("Content-Type", "text/event-stream")
			for i := 0; ; i++ {
				d, _ := json.Marshal(map[string]string{"message": "line\n", "cursor": fmt.Sprint(i)})
				if _, err := fmt.Fprintf(w, "event: message\ndata: %s\n\n", d); err != nil {
					return
				}
				w.(http.Flusher).Flush()
				select {
				case <-r.Context().Done():
					return
				case <-time.After(20 * time.Millisecond):
				}
			}
		},
		"stderr": finishedLog(),
	})
	cfg.Timeout = 500 * time.Millisecond
	err := waitForJob(context.Background(), cfg, io.Discard, io.Discard, "job1")
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected a timeout error, got %v", err)
	}
}
