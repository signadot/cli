package sandboxmanager

import (
	"context"
	"sync"
	"time"

	"github.com/signadot/go-sdk/models"
	clapi "github.com/signadot/libconnect/apiv1"
	clapiclient "github.com/signadot/libconnect/common/apiclient"
	"github.com/signadot/libconnect/revtun"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	reconcilePeriod = 10 * time.Second
)

type sbMonitor struct {
	sync.Mutex
	routingKey   string
	clapiClient  clapiclient.Client
	revtunClient revtun.Client
	// func called on delete
	delFn       func()
	log         *slog.Logger
	doneCh      chan struct{}
	reconcileCh chan struct{}
	status      *clapi.WatchSandboxStatus
	revtuns     map[string]*rt
	locals      map[string]*models.Local
}

func newSBMonitor(rk string, clapiClient clapiclient.Client, rtClient revtun.Client, delFn func(), log *slog.Logger) *sbMonitor {
	res := &sbMonitor{
		routingKey:   rk,
		clapiClient:  clapiClient,
		revtunClient: rtClient,
		delFn:        delFn,
		log:          log,
		doneCh:       make(chan struct{}),
		reconcileCh:  make(chan struct{}, 1),
		locals:       make(map[string]*models.Local),
		revtuns:      make(map[string]*rt),
	}
	go res.monitor()
	return res
}

func (sbm *sbMonitor) getStatus() *clapi.WatchSandboxStatus {
	sbm.Lock()
	defer sbm.Unlock()
	return sbm.status
}

func (sbm *sbMonitor) stop() {
	select {
	case sbm.doneCh <- struct{}{}:
	default:
	}
}

func (sbm *sbMonitor) monitor() {
	// setup context for grp stream requests
	ctx, cancel := context.WithCancel(context.Background())

	// watch the given sandbox
	go sbm.watchSandbox(ctx)

	// run the reconcile loop
	ticker := time.NewTicker(reconcilePeriod)
	defer ticker.Stop()
reconcileLoop:
	for {
		select {
		case <-sbm.doneCh:
			// we are done, cancel the context
			cancel()
			break reconcileLoop
		case <-sbm.reconcileCh:
			// The status has changed
			sbm.reconcile()
		case <-ticker.C:
			// Reconcile ticker
			sbm.reconcile()
		}
	}

	// we're done, clean up revtuns and parent delete func
	sbm.log.Debug("cleaning up status and locals and parent")
	sbm.updateSandboxStatus(&clapi.WatchSandboxStatus{})
	sbm.updateLocalsSpec(nil)
	sbm.reconcile()
	sbm.delFn()
}

func (sbm *sbMonitor) watchSandbox(ctx context.Context) {
	// watch loop
	for {
		sbwClient, err := sbm.clapiClient.WatchSandbox(ctx, &clapi.WatchSandboxRequest{
			RoutingKey: sbm.routingKey,
		})
		if err != nil {
			sbm.log.Error("error getting sb watch stream, retrying", "error", err)
			<-time.After(3 * time.Second)
			continue
		}
		sbm.log.Debug("successfully got sandbox watch client")
		err = sbm.readStream(sbwClient)
		if err == nil {
			// NotFound
			break
		}
	}

	// There is no sandbox, stop the monitor
	sbm.stop()
}

func (sbm *sbMonitor) readStream(sbwClient clapi.TunnelAPI_WatchSandboxClient) error {
	var (
		ok         bool
		err        error
		sbStatus   *clapi.WatchSandboxStatus
		grpcStatus *status.Status
	)
	for {
		sbStatus, err = sbwClient.Recv()
		if err == nil {
			sbm.updateSandboxStatus(sbStatus)
			continue
		}
		if grpcStatus, ok = status.FromError(err); !ok {
			sbm.log.Error("sandbox monitor grpc stream error: no status",
				"error", err)
			break
		}
		switch grpcStatus.Code() {
		case codes.OK:
			sbm.log.Debug("sandbox watch stream error code is ok")
			sbm.updateSandboxStatus(sbStatus)
			continue
		case codes.Internal:
			sbm.log.Error("sandbox watch: internal grpc error",
				"error", err)
		case codes.NotFound:
			sbm.log.Info("sandbox watch: sandbox not found")
			err = nil
		default:
			sbm.log.Error("sandbox watch error", "error", err)
		}
		break
	}
	return err
}

func (sbm *sbMonitor) updateSandboxStatus(st *clapi.WatchSandboxStatus) {
	sbm.Lock()
	defer sbm.Unlock()

	// update status
	sbm.status = st
	sbm.log.Debug("sbm setting watch status", "status", sbm.status)
	// trigger a reconcile
	sbm.triggerReconcile()
}

func (sbm *sbMonitor) updateLocalsSpec(locals []*models.Local) {
	sbm.Lock()
	defer sbm.Unlock()

	localMap := make(map[string]*models.Local, len(locals))
	for _, localSpec := range locals {
		localMap[localSpec.Name] = localSpec
	}
	for localName := range sbm.locals {
		_, desired := localMap[localName]
		if !desired {
			delete(sbm.locals, localName)
			continue
		}
	}
	for localName, local := range localMap {
		_, has := sbm.locals[localName]
		if has {
			// TODO check if local def changed, if so, close
			// old revtun and create new one
			continue
		}
		sbm.locals[localName] = local
	}

	// trigger a reconcile
	sbm.triggerReconcile()
}

func (sbm *sbMonitor) triggerReconcile() {
	select {
	case sbm.reconcileCh <- struct{}{}:
	default:
	}
}

func (sbm *sbMonitor) reconcile() {
	sbm.Lock()
	defer sbm.Unlock()
	if sbm.status == nil {
		return
	}

	// reconcile
	sbm.log.Debug("reconciling tunnels")
	statusXWs := make(map[string]*clapi.WatchSandboxStatus_ExternalWorkload, len(sbm.status.ExternalWorkloads))
	// put the xwls in a map
	for _, xw := range sbm.status.ExternalWorkloads {
		statusXWs[xw.Name] = xw
	}

	for xwName, sxw := range statusXWs {
		rt, has := sbm.revtuns[xwName]
		if has {
			select {
			case <-rt.rtClosed:
				has = false
				delete(sbm.revtuns, xwName)
			case <-rt.rtToClose:
				has = false
				delete(sbm.revtuns, xwName)
			default:
			}
		}

		if has {
			// check connection issues
			if !sxw.Connected {
				now := time.Now()
				if rt.clusterNotConnectedTime == nil {
					rt.clusterNotConnectedTime = &now
				}
				if time.Since(*rt.clusterNotConnectedTime) > 10*time.Second {
					has = false
					select {
					case <-rt.rtToClose:
					default:
						close(rt.rtToClose)
					}
					delete(sbm.revtuns, xwName)
				}
			} else {
				// reset this value
				rt.clusterNotConnectedTime = nil
			}
		}
		if has {
			continue
		}

		// get local spec
		local := sbm.locals[xwName]
		if local == nil {
			sbm.log.Warn("no local found for cluster extworkload status", "local", xwName)
			continue
		}

		// create revtun
		rt, err := newRevtun(sbm.log.With("local", sxw.Name),
			sbm.revtunClient, sxw.Name, sbm.routingKey, local)
		if err != nil {
			sbm.log.Error("error creating revtun", "error", err)
			continue
		}
		sbm.revtuns[xwName] = rt
	}

	// delete unwanted tunnels
	for xwName, xw := range sbm.revtuns {
		_, desired := statusXWs[xwName]
		if desired {
			continue
		}
		select {
		case <-xw.rtToClose:
		default:
			sbm.log.Debug("sandbox monitor closing revtun", "local", xwName)
			close(xw.rtToClose)
			delete(sbm.revtuns, xwName)
		}
	}
}
