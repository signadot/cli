package local

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/signadot/cli/internal/config"
)

func TestReportDisconnectWait(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, config.RootManagerPIDFile), []byte("4242"), 0600); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	reportDisconnectWait(&b, dir, &runState{RootPIDFilePresent: true}, 7*time.Second, errors.New("connection refused"))
	out := b.String()
	for _, want := range []string{"7s", "connection refused", config.RootManagerPIDFile, "pid 4242", "stale"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, config.SandboxManagerPIDFile) {
		t.Errorf("output mentions absent sandbox manager pidfile:\n%s", out)
	}
}
