package sandboxmanager

import (
	"context"
	"log/slog"
	"sync"
	"time"

	tunapiv1 "github.com/signadot/libconnect/apiv1"
	tunapiclient "github.com/signadot/libconnect/common/apiclient"
	"github.com/signadot/libconnect/revtun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type sbmWatcher struct {
	log *slog.Logger

	machineID string

	// sandbox controllers
	sbMu          sync.Mutex
	sbControllers map[string]*sbController
	revtunClient  func() revtun.Client

	// shutdown
	shutdownCh chan struct{}
}

func newSandboxManagerWatcher(log *slog.Logger, machineID string, revtunClient func() revtun.Client,
	shutdownCh chan struct{}) *sbmWatcher {
	srv := &sbmWatcher{
		log:           log,
		machineID:     machineID,
		sbControllers: make(map[string]*sbController),
		revtunClient:  revtunClient,
		shutdownCh:    shutdownCh,
	}
	return srv
}

func (sbw *sbmWatcher) run(ctx context.Context, tunAPIClient tunapiclient.Client) {
	go sbw.watchSandboxes(ctx, tunAPIClient)
}

func (sbw *sbmWatcher) watchSandboxes(ctx context.Context, tunAPIClient tunapiclient.Client) {
	// watch loop
	for {
		sbwClient, err := tunAPIClient.WatchLocalSandboxes(ctx, &tunapiv1.WatchLocalSandboxesRequest{
			MachineId: sbw.machineID,
		})
		if err != nil {
			// don't retry if the context has been cancelled
			select {
			case <-ctx.Done():
				return
			default:
			}

			sbw.log.Error("error getting local sandboxes watch stream, retrying", "error", err)
			<-time.After(3 * time.Second)
			continue
		}
		sbw.log.Debug("successfully got local sandboxes watch client")
		err = sbw.readStream(sbwClient)
		if err == nil {
			// NotFound
			break
		}
	}
}

func (sbw *sbmWatcher) readStream(sbwClient tunapiv1.TunnelAPI_WatchLocalSandboxesClient) error {
	var (
		ok         bool
		err        error
		event      *tunapiv1.WatchLocalSandboxesResponse
		grpcStatus *status.Status
	)
	for {
		event, err = sbwClient.Recv()
		if err == nil {
			sbw.processStreamEvent(event)
			continue
		}
		if grpcStatus, ok = status.FromError(err); !ok {
			sbw.log.Error("sandboxes watch grpc stream error: no status",
				"error", err)
			break
		}
		switch grpcStatus.Code() {
		case codes.OK:
			sbw.log.Debug("sandboxes watch error code is ok")
			sbw.processStreamEvent(event)
			continue
		case codes.Internal:
			sbw.log.Error("sandboxes watch internal grpc error",
				"error", err)
			<-time.After(3 * time.Second)
		default:
			sbw.log.Error("sandbox watch error", "error", err)
		}
		break
	}
	return err
}

func (sbw *sbmWatcher) processStreamEvent(event *tunapiv1.WatchLocalSandboxesResponse) {
	sbw.sbMu.Lock()
	defer sbw.sbMu.Unlock()

	desiredSandboxes := map[string]bool{}
	for i := range event.Sandboxes {
		sds := event.Sandboxes[i]
		desiredSandboxes[sds.SandboxName] = true

		if ctrl := sbw.sbControllers[sds.SandboxName]; ctrl != nil {
			// update the sandbox in the controller
			ctrl.updateSandbox(sds)
		} else {
			// create a new sandbox controller
			sbw.log.Debug("creating sandbox", "sandbox", sds)
			sbw.sbControllers[sds.SandboxName] = newSBController(
				sbw.log, sds, sbw.revtunClient(),
				func() {
					sbw.sbMu.Lock()
					defer sbw.sbMu.Unlock()
					delete(sbw.sbControllers, sds.SandboxName)
				},
			)
		}
	}

	// remove unwanted sandboxes
	for sdsName, ctrl := range sbw.sbControllers {
		if _, ok := desiredSandboxes[sdsName]; ok {
			continue
		}
		sbw.log.Debug("removing sandbox", "sandboxName", sdsName)
		ctrl.stop()
	}
}

func (sbw *sbmWatcher) getSandboxes() []*tunapiv1.WatchLocalSandboxesResponse_Sandbox {
	sbw.sbMu.Lock()
	defer sbw.sbMu.Unlock()

	res := make([]*tunapiv1.WatchLocalSandboxesResponse_Sandbox, 0, len(sbw.sbControllers))
	for _, ctrl := range sbw.sbControllers {
		res = append(res, ctrl.getSandbox())
	}
	return res
}

func (sbw *sbmWatcher) stop() {
	var wg sync.WaitGroup

	// stop all sandbox controllers
	sbw.sbMu.Lock()
	for _, sbMon := range sbw.sbControllers {
		wg.Add(1)
		// overwrite the delete function
		sbMon.delFn = func() {
			defer wg.Done()
		}
		// stop the sandbox controller
		sbMon.stop()
	}
	sbw.sbMu.Unlock()

	// wait until all sandbox controllers have stopped
	wg.Wait()
}
