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
				Attempts: []*models.JobsAttempt{{Phase: phase}},
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
