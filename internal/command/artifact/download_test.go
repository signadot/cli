package artifact

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/signadot/cli/internal/config"
	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"
)

func fakeDownload(t *testing.T, h http.HandlerFunc) *config.ArtifactDownload {
	t.Helper()
	keyring.MockInit()
	t.Setenv("HOME", t.TempDir())
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	viper.Set("api_key", "key")
	viper.Set("org", "org")
	viper.Set("api_url", srv.URL)
	t.Cleanup(viper.Reset)
	return &config.ArtifactDownload{
		Artifact: &config.Artifact{API: &config.API{}},
		Job:      "job1",
	}
}

func dirEntries(t *testing.T, dir string) []string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range es {
		names = append(names, e.Name())
	}
	return names
}

// A failed download must leave an existing file untouched and no temp file.
func TestDownloadErrorKeepsExistingFile(t *testing.T) {
	cfg := fakeDownload(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})
	dir := t.TempDir()
	cfg.OutputFile = filepath.Join(dir, "report.xml")
	if err := os.WriteFile(cfg.OutputFile, []byte("precious"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := download(context.Background(), cfg, io.Discard, "report.xml"); err == nil {
		t.Fatal("expected error")
	}
	if got, _ := os.ReadFile(cfg.OutputFile); string(got) != "precious" {
		t.Fatalf("existing file changed to %q", got)
	}
	if names := dirEntries(t, dir); len(names) != 1 {
		t.Fatalf("unexpected files left: %v", names)
	}
}

func TestDownloadWritesFile(t *testing.T) {
	cfg := fakeDownload(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/orgs/org/artifacts/jobs/job1/attempts/0/objects/download" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		io.WriteString(w, "content")
	})
	dir := t.TempDir()
	cfg.OutputFile = filepath.Join(dir, "report.xml")
	if err := os.WriteFile(cfg.OutputFile, []byte("old"), 0640); err != nil {
		t.Fatal(err)
	}
	if err := download(context.Background(), cfg, io.Discard, "report.xml"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(cfg.OutputFile)
	if string(got) != "content" {
		t.Fatalf("file content = %q", got)
	}
	if fi, _ := os.Stat(cfg.OutputFile); fi.Mode().Perm() != 0640 {
		t.Errorf("mode = %v, want existing 0640 kept", fi.Mode().Perm())
	}
	if names := dirEntries(t, dir); len(names) != 1 {
		t.Fatalf("unexpected files left: %v", names)
	}
}
