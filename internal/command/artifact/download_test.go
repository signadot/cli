package artifact

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

// A download that fails before any data must leave an existing file untouched.
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

func serveArtifact(content string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/orgs/org/artifacts/jobs/job1/attempts/0/objects/download" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		io.WriteString(w, content)
	}
}

// -o names the file the output goes to, so a symlink is written through: the
// link stays a link and its target gets the content.
func TestDownloadWritesThroughASymlink(t *testing.T) {
	cfg := fakeDownload(t, serveArtifact("content"))
	dir := t.TempDir()
	real := filepath.Join(dir, "real.xml")
	if err := os.WriteFile(real, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg.OutputFile = filepath.Join(dir, "link.xml")
	if err := os.Symlink(real, cfg.OutputFile); err != nil {
		t.Fatal(err)
	}
	if err := download(context.Background(), cfg, io.Discard, "report.xml"); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(cfg.OutputFile); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("the link was replaced: %v %v", fi.Mode(), err)
	}
	if got, _ := os.ReadFile(real); string(got) != "content" {
		t.Fatalf("link target = %q, want the download", got)
	}
}

// An empty artifact is an empty file, not no file.
func TestDownloadEmptyArtifactCreatesFile(t *testing.T) {
	cfg := fakeDownload(t, serveArtifact(""))
	cfg.OutputFile = filepath.Join(t.TempDir(), "empty.txt")
	if err := download(context.Background(), cfg, io.Discard, "empty.txt"); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(cfg.OutputFile); err != nil || fi.Size() != 0 {
		t.Fatalf("want an empty file, got %v %v", fi, err)
	}
}

// A download that stops sending is an error once it has been silent for the
// idle timeout, rather than a hang; what arrived is kept.
func TestDownloadStalledIsAnError(t *testing.T) {
	old := downloadIdleTimeout
	downloadIdleTimeout = 200 * time.Millisecond
	t.Cleanup(func() { downloadIdleTimeout = old })
	cfg := fakeDownload(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		io.WriteString(w, "partial")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	})
	cfg.OutputFile = filepath.Join(t.TempDir(), "big.bin")
	err := download(context.Background(), cfg, io.Discard, "big.bin")
	if err == nil || !strings.Contains(err.Error(), "stalled") {
		t.Fatalf("want a stall error, got %v", err)
	}
	if got, _ := os.ReadFile(cfg.OutputFile); string(got) != "partial" {
		t.Errorf("file = %q, want what arrived", got)
	}
}
