package rootmanager

import (
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/signadot/cli/internal/config"
)

// If the sandbox manager command can't be formulated (e.g. the binary was
// moved while connected), run returns; stop must still complete so that
// shutdown can restore networking.
func TestSBMgrMonitorStopAfterFatalRunError(t *testing.T) {
	orig := os.Args[0]
	os.Args[0] = "/nonexistent/signadot"
	defer func() { os.Args[0] = orig }()

	mon := newSBMgrMonitor(&config.ConnectInvocationConfig{
		SignadotDir: t.TempDir(),
		User:        &config.ConnectInvocationUser{},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	mon.run() // returns: binary not found

	stopped := make(chan error, 1)
	go func() { stopped <- mon.stop() }()
	select {
	case err := <-stopped:
		if err != nil {
			t.Fatalf("stop: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("stop blocked after run exited on a fatal error")
	}
}
