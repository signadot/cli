package jobs

import (
	"context"
	"encoding/json"
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
)

func fakeJobConfig(t *testing.T, phase models.JobsPhase) *config.JobSubmit {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	return &config.JobSubmit{
		Job:     &config.Job{API: &config.API{Org: "org", Client: c}},
		Timeout: 10 * time.Second,
	}
}

func TestWaitForJobCanceledIsError(t *testing.T) {
	cfg := fakeJobConfig(t, models.JobsPhaseCanceled)
	err := waitForJob(context.Background(), cfg, io.Discard, io.Discard, "job1")
	if err == nil || !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("expected canceled error, got %v", err)
	}
}

func TestWaitForJobSucceeded(t *testing.T) {
	cfg := fakeJobConfig(t, models.JobsPhaseSucceeded)
	if err := waitForJob(context.Background(), cfg, io.Discard, io.Discard, "job1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
