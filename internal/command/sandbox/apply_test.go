package sandbox

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"
)

func fakeApplyConfig(t *testing.T, sandboxFile string, h http.HandlerFunc) *config.SandboxApply {
	t.Helper()
	keyring.MockInit()
	t.Setenv("HOME", t.TempDir())
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	viper.Set("api_key", "key")
	viper.Set("org", "org")
	viper.Set("api_url", srv.URL)
	t.Cleanup(viper.Reset)

	f := filepath.Join(t.TempDir(), "sb.yaml")
	if err := os.WriteFile(f, []byte(sandboxFile), 0644); err != nil {
		t.Fatal(err)
	}
	return &config.SandboxApply{
		Sandbox: &config.Sandbox{API: &config.API{
			Root: config.Root{DashboardURL: &url.URL{Scheme: "https", Host: "app.example.com"}},
		}},
		Filename: f,
	}
}

func TestApplyMissingSpec(t *testing.T) {
	cfg := fakeApplyConfig(t, "name: sb\n", http.NotFound)
	err := apply(cfg, io.Discard, io.Discard, nil)
	if err == nil || !strings.Contains(err.Error(), "spec") {
		t.Fatalf("got %v, want missing spec error", err)
	}
}

// If waiting fails before any successful GET, the applied sandbox is output
// (instead of panicking on a nil sandbox) and the error returned.
func TestApplyWaitFailsBeforeAnyGet(t *testing.T) {
	cfg := fakeApplyConfig(t, "name: sb\nspec:\n  cluster: c\n", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "unavailable", http.StatusInternalServerError)
			return
		}
		cluster := "c"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&models.Sandbox{
			Name: "sb", RoutingKey: "rk", Spec: &models.SandboxSpec{Cluster: &cluster},
		})
	})
	cfg.Wait = true
	cfg.WaitTimeout = time.Millisecond
	var out strings.Builder
	if err := apply(cfg, &out, io.Discard, nil); err == nil {
		t.Fatal("expected wait error")
	}
	if !strings.Contains(out.String(), "Dashboard page") {
		t.Errorf("applied sandbox not output: %q", out.String())
	}
}
