package plan

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"
)

func TestExportOutputs(t *testing.T) {
	keyring.MockInit()
	t.Setenv("HOME", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/orgs/org/plans/executions/ex1/outputs/good":
			w.Header().Set("Content-Type", "application/octet-stream")
			io.WriteString(w, "good content")
		default:
			http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
		}
	}))
	defer srv.Close()
	viper.Set("api_key", "key")
	viper.Set("org", "org")
	viper.Set("api_url", srv.URL)
	defer viper.Reset()

	dir := t.TempDir()
	cfg := &config.PlanRun{Plan: &config.Plan{API: &config.API{}}, OutputDir: dir}
	if err := cfg.InitAPIConfig(); err != nil {
		t.Fatal(err)
	}
	exec := &models.PlanExecution{
		ID: "ex1",
		Status: &models.PlanExecutionStatus{Outputs: []*models.PlanOutputStatus{
			{Name: "good"}, {Name: "bad"},
		}},
	}
	err := exportOutputs(context.Background(), cfg, io.Discard, exec)
	if err == nil {
		t.Fatal("expected error for the failed output")
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "good")); string(got) != "good content" {
		t.Errorf("good output = %q", got)
	}
	es, _ := os.ReadDir(dir)
	if len(es) != 1 {
		var names []string
		for _, e := range es {
			names = append(names, e.Name())
		}
		t.Errorf("unexpected files: %v", names)
	}
}
