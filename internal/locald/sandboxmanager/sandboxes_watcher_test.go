package sandboxmanager

import (
	"context"
	"io"
	"log/slog"
	"testing"

	sbapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Before (or without) a healthy link to the cluster there is no tunnel API
// client: GetResourceOutputs must fail cleanly instead of crashing the
// sandbox manager, and running the watcher must not panic.
func TestNoTunnelAPIClient(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	sbw := newSandboxManagerWatcher(log, "", nil, &operatorInfoUpdater{log: log}, make(chan struct{}))
	srv := &sbmServer{log: log, sbmWatcher: sbw}

	_, err := srv.GetResourceOutputs(context.Background(), &sbapi.GetResourceOutputsRequest{SandboxRoutingKey: "rk"})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("got %v, want Unavailable", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sbw.run(ctx, nil)
	sbw.stop()
}
