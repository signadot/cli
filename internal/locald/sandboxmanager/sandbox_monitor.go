package sandboxmanager

import (
	"bytes"
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
	status      *clapi.WatchSandboxResponse
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

func (sbm *sbMonitor) getStatus() *clapi.WatchSandboxResponse {
	sbm.Lock()
	defer sbm.Unlock()
	return sbm.status
}

func (sbm *sbMonitor) stop() {
	select {
	case <-sbm.doneCh:
	default:
		close(sbm.doneCh)
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
	sbm.updateSandboxStatus(&clapi.WatchSandboxResponse{})
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
			// don't retry if the context has been cancelled
			select {
			case <-ctx.Done():
				return
			default:
			}

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
		sbStatus   *clapi.WatchSandboxResponse
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
			<-time.After(3 * time.Second)
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

func (sbm *sbMonitor) updateSandboxStatus(st *clapi.WatchSandboxResponse) {
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

	desLocals := make(map[string]*models.Local, len(locals))
	for _, localSpec := range locals {
		desLocals[localSpec.Name] = localSpec
	}
	for localName := range sbm.locals {
		_, desired := desLocals[localName]
		if !desired {
			delete(sbm.locals, localName)
			continue
		}
	}
	for localName, des := range desLocals {
		obs, has := sbm.locals[localName]
		if !has {
			sbm.locals[localName] = des
			continue
		}
		if !sbm.localsEqual(des, obs) {
			sbm.log.Debug("not equal", "des", des, "obs", obs)
			sbm.closeRevTunnel(localName)
		}
		sbm.locals[localName] = des
	}

	// trigger a reconcile
	sbm.triggerReconcile()
}

func (sbm *sbMonitor) closeRevTunnel(xwName string) {
	revtun := sbm.revtuns[xwName]
	if revtun == nil {
		return
	}

	select {
	case <-revtun.rtToClose:
	default:
		sbm.log.Debug("sandbox monitor closing revtun", "local", xwName)
		close(revtun.rtToClose)
	}
}

func (sbm *sbMonitor) localsEqual(a, b *models.Local) bool {
	da, err := a.MarshalBinary()
	if err != nil {
		sbm.log.Error("error marshalling local", "error", err)
		return false
	}
	db, err := b.MarshalBinary()
	if err != nil {
		sbm.log.Error("error marshalling local", "error", err)
		return false
	}
	return bytes.Equal(da, db)
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
	statusXWs := make(map[string]*clapi.WatchSandboxResponse_ExternalWorkload, len(sbm.status.ExternalWorkloads))
	// put the xwls in a map
	for _, xw := range sbm.status.ExternalWorkloads {
		statusXWs[xw.Name] = xw
	}

	for xwName, sxw := range statusXWs {
		rt, has := sbm.revtuns[xwName]
		if has {
			select {
			case <-rt.rtToClose:
				// delete the revtun if it has been closed
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
					// this revtun has been down for more than 10 secs
					// close and delete the current revtune
					has = false
					sbm.closeRevTunnel(xwName)
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
	for xwName := range sbm.revtuns {
		_, desired := statusXWs[xwName]
		if desired {
			continue
		}
		// close and delete the current revtune
		sbm.closeRevTunnel(xwName)
		delete(sbm.revtuns, xwName)
	}
}
