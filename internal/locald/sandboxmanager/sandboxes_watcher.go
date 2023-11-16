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
	oiu *operatorInfoUpdater

	machineID string

	// sandbox controllers
	sbMu          sync.Mutex
	status        svchealth.ServiceHealth
	sbControllers map[string]*sbController
	sbMonitors    map[string]*sbMonitor
	revtunClient  func() revtun.Client
	tunAPIClient  tunapiclient.Client

	// shutdown
	shutdownCh chan struct{}
}

func newSandboxManagerWatcher(log *slog.Logger, machineID string, revtunClient func() revtun.Client,
	oiu *operatorInfoUpdater, shutdownCh chan struct{}) *sbmWatcher {
	srv := &sbmWatcher{
		log:       log,
		oiu:       oiu,
		machineID: machineID,
		status: svchealth.ServiceHealth{
			Healthy:         false,
			LastErrorReason: "Starting",
		},
		sbControllers: map[string]*sbController{},
		sbMonitors:    map[string]*sbMonitor{},
		revtunClient:  revtunClient,
		shutdownCh:    shutdownCh,
	}
	return srv
}

func (sbw *sbmWatcher) run(ctx context.Context, tunAPIClient tunapiclient.Client) {
	sbw.tunAPIClient = tunAPIClient
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
		sbw.readStream(ctx, sbwClient)
	}
}

func (sbw *sbmWatcher) readStream(ctx context.Context,
	sbwClient tunapiv1.TunnelAPI_WatchLocalSandboxesClient) {
	for {
		event, err := sbwClient.Recv()
		if err == nil {
			sbw.setSuccess(ctx)
			sbw.processStreamEvent(event)
			continue
		}
		// just return if the context has been cancelled
		select {
		case <-ctx.Done():
			return
		default:
		}
		// extract the grpc status
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
			sbw.setError("this feature requires operator >= 0.14.1", nil)
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
		// update the sds object
		sbw.processSDS(sds)
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

func (sbw *sbmWatcher) processSDS(sds *tunapiv1.Sandbox) {
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

	sbw.sbMu.Lock()
	// stop all sandbox monitor (if any)
	sbw.stopMonitors()

	// stop all sandbox controllers:
	//
	// - the grpcserver has called graceful stop, so no new sandboxes will be
	// processed during a call to this function
	// - the delFn is called at the end of the sb controller run, so the wg will
	// wait for all monitors to stop running.
	//
	for _, ctrl := range sbw.sbControllers {
		wg.Add(1)
		// overwrite the delete function
		ctrl.delFn = func() {
			defer wg.Done()
		}
		// stop the sandbox controller
		ctrl.stop()
	}
	sbw.sbMu.Unlock()

	// wait until all sandbox controllers have stopped
	wg.Wait()
}

// This function does not perform any locking, thus so the lock should acquired
// by the caller.
func (sbw *sbmWatcher) stopMonitors() {
	for sandboxName, sbm := range sbw.sbMonitors {
		sbm.stop()
		delete(sbw.sbMonitors, sandboxName)
	}
}

func (sbw *sbmWatcher) setSuccess(ctx context.Context) {
	sbw.sbMu.Lock()
	defer sbw.sbMu.Unlock()

	// update the status
	sbw.status.Healthy = true

	// try loading the operator info
	sbw.oiu.Reload(ctx, sbw.tunAPIClient, false)

	// stop all sandbox monitor (if any)
	sbw.stopMonitors()
}

func (sbw *sbmWatcher) setError(errMsg string, err error) {
	sbw.sbMu.Lock()
	defer sbw.sbMu.Unlock()

	// update the status
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

	// reset the operator info
	sbw.oiu.Reset()
}

func (sbw *sbmWatcher) registerSandbox(sandboxName, routingKey string) {
	sbw.sbMu.Lock()
	defer sbw.sbMu.Unlock()

	if sbw.status.Healthy {
		// the sandbox watcher is running, no need to register this sandbox (it
		// should happen via WatchLocalSandboxes stream)
		return
	}

	// create a sandbox monitor (that will watch events specific to that sandbox)
	if sbm := sbw.sbMonitors[sandboxName]; sbm != nil {
		sbm.stop()
	}
	sbw.sbMonitors[sandboxName] = newSBMonitor(sbw.log, sandboxName, routingKey,
		sbw.tunAPIClient, sbw.monitorUpdatedSandbox)
}

func (sbw *sbmWatcher) monitorUpdatedSandbox(routingKey string, event *tunapiv1.WatchSandboxResponse) {
	sbw.sbMu.Lock()
	defer sbw.sbMu.Unlock()

	if _, ok := sbw.sbMonitors[event.SandboxName]; !ok {
		// this could only happen while the sandbox monitor is being deleted,
		// just ignore this event
		return
	}

	if event == nil {
		// the sandbox has been deleted, the monitor will automatically stop,
		// lets stop its controller here
		ctrl := sbw.sbControllers[event.SandboxName]
		if ctrl != nil {
			sbw.log.Debug("removing sandbox", "sandboxName", event.SandboxName)
			ctrl.stop()
		}

		// remove the reference to the monitor
		delete(sbw.sbMonitors, event.SandboxName)
		return
	}

	// a sandbox has been added/updated
	sds := &tunapiv1.Sandbox{
		SandboxName:       event.SandboxName,
		RoutingKey:        routingKey,
		ExternalWorkloads: event.ExternalWorkloads,
		Resources:         event.Resources,
	}
	sbw.processSDS(sds)
}
