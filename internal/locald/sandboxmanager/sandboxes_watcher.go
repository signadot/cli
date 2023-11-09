package sandboxmanager

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	tunapiv1 "github.com/signadot/libconnect/apiv1"
	tunapiclient "github.com/signadot/libconnect/common/apiclient"
	"github.com/signadot/libconnect/common/svchealth"
	"github.com/signadot/libconnect/revtun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type sbmWatcher struct {
	log *slog.Logger

	machineID string

	// sandbox controllers
	sbMu          sync.Mutex
	status        svchealth.ServiceHealth
	sbControllers map[string]*sbController
	revtunClient  func() revtun.Client

	// shutdown
	shutdownCh chan struct{}
}

func newSandboxManagerWatcher(log *slog.Logger, machineID string, revtunClient func() revtun.Client,
	shutdownCh chan struct{}) *sbmWatcher {
	srv := &sbmWatcher{
		log:       log,
		machineID: machineID,
		status: svchealth.ServiceHealth{
			Healthy:         false,
			LastErrorReason: "Starting",
		},
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

			sbw.setError("error getting local sandboxes watch stream", err)
			<-time.After(3 * time.Second)
			continue
		}
		sbw.log.Debug("successfully got local sandboxes watch client")
		sbw.readStream(sbwClient)
	}
}

func (sbw *sbmWatcher) readStream(sbwClient tunapiv1.TunnelAPI_WatchLocalSandboxesClient) {
	for {
		event, err := sbwClient.Recv()
		if err == nil {
			sbw.setSuccess()
			sbw.processStreamEvent(event)
			continue
		}
		grpcStatus, ok := status.FromError(err)
		if !ok {
			sbw.setError("sandboxes watch grpc stream error: no status", err)
			break
		}
		switch grpcStatus.Code() {
		case codes.OK:
			sbw.log.Debug("sandboxes watch error code is ok")
			sbw.processStreamEvent(event)
			continue
		case codes.Internal:
			sbw.setError("sandboxes watch internal grpc error", err)
			<-time.After(3 * time.Second)
		case codes.Unimplemented:
			sbw.setError("incompatible operator version, current CLI requires operator >= 0.15.0", nil)
			// in this case, check again in 1 minutes
			<-time.After(1 * time.Minute)
		default:
			sbw.setError("sandbox watch error", err)
			<-time.After(3 * time.Second)
		}
		break
	}
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

func (sbw *sbmWatcher) getStatus() *svchealth.ServiceHealth {
	sbw.sbMu.Lock()
	defer sbw.sbMu.Unlock()

	return &sbw.status
}

func (sbw *sbmWatcher) getSandboxes() []*tunapiv1.Sandbox {
	sbw.sbMu.Lock()
	defer sbw.sbMu.Unlock()

	res := make([]*tunapiv1.Sandbox, 0, len(sbw.sbControllers))
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

func (sbw *sbmWatcher) setSuccess() {
	sbw.sbMu.Lock()
	defer sbw.sbMu.Unlock()

	sbw.status.Healthy = true
}

func (sbw *sbmWatcher) setError(errMsg string, err error) {
	sbw.sbMu.Lock()
	defer sbw.sbMu.Unlock()

	now := time.Now()
	var reason string
	if err != nil {
		reason = fmt.Sprintf("%s, %s", errMsg, err.Error())
		sbw.log.Error(errMsg, "error", err)
	} else {
		reason = errMsg
		sbw.log.Error(errMsg)
	}

	sbw.status.Healthy = false
	sbw.status.LastErrorReason = reason
	sbw.status.LastErrorTime = &now
	sbw.status.ErrorCount += 1
}
