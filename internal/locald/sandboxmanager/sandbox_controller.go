package sandboxmanager

import (
	"sync"
	"time"

	"log/slog"

	"github.com/signadot/libconnect/apiv1"
	tunapiv1 "github.com/signadot/libconnect/apiv1"
	"github.com/signadot/libconnect/revtun"
	"google.golang.org/protobuf/proto"
)

const (
	reconcilePeriod = 10 * time.Second
)

type sbController struct {
	sync.Mutex

	log          *slog.Logger
	sandbox      *tunapiv1.Sandbox
	revtunClient revtun.Client
	revtuns      map[string]*rt
	delFn        func()

	reconcileCh chan struct{}
	doneCh      chan struct{}
}

func newSBController(log *slog.Logger, sandbox *tunapiv1.Sandbox,
	rtClient revtun.Client, delFn func()) *sbController {
	// create the controller
	ctrl := &sbController{
		log:          log.With("sandbox", sandbox.SandboxName),
		sandbox:      sandbox,
		revtunClient: rtClient,
		revtuns:      make(map[string]*rt),
		delFn:        delFn,
		reconcileCh:  make(chan struct{}, 1),
		doneCh:       make(chan struct{}),
	}
	// run the controller
	go ctrl.run()
	return ctrl
}

func (ctrl *sbController) run() {
	// trigger a reconcile
	ctrl.triggerReconcile()

	// run the reconcile loop
	ticker := time.NewTicker(reconcilePeriod)
	defer ticker.Stop()
reconcileLoop:
	for {
		select {
		case <-ctrl.doneCh:
			// we are done, cancel the context
			break reconcileLoop
		case <-ctrl.reconcileCh:
			// The status has changed
			ctrl.reconcile()
		case <-ticker.C:
			// Reconcile ticker
			ctrl.reconcile()
		}
	}

	// we're done, clean up revtuns
	ctrl.log.Debug("cleaning up reverse tunnels")
	ctrl.updateSandbox(&tunapiv1.Sandbox{})
	ctrl.reconcile()
	ctrl.delFn()
}

func (ctrl *sbController) stop() {
	select {
	case <-ctrl.doneCh:
	default:
		close(ctrl.doneCh)
	}
}

func (ctrl *sbController) getSandbox() *tunapiv1.Sandbox {
	ctrl.Lock()
	defer ctrl.Unlock()
	return ctrl.sandbox
}

func (ctrl *sbController) updateSandbox(sandbox *tunapiv1.Sandbox) {
	ctrl.Lock()
	defer ctrl.Unlock()

	// close rev tunnels if the spec has changed
	for _, desXW := range sandbox.ExternalWorkloads {
		for _, obsXW := range ctrl.sandbox.ExternalWorkloads {
			if desXW.Name != obsXW.Name {
				continue
			}
			if !ctrl.compareExternalWorkloadsSpec(desXW, obsXW) {
				// the desired local spec is different from the observed one,
				// close existing tunnel
				ctrl.log.Debug("desired local spec is different from the observed one", "des", desXW, "obs", obsXW)
				ctrl.closeRevTunnel(obsXW.Name)
			}
			break
		}
	}

	// update sandbox
	ctrl.sandbox.ExternalWorkloads = sandbox.ExternalWorkloads
	ctrl.sandbox.Resources = sandbox.Resources
	ctrl.log.Debug("updating sandbox", "sandbox", ctrl.sandbox)

	// trigger a reconcile
	ctrl.triggerReconcile()
}

func (ctrl *sbController) triggerReconcile() {
	select {
	case ctrl.reconcileCh <- struct{}{}:
	default:
	}
}

func (ctrl *sbController) reconcile() {
	ctrl.Lock()
	defer ctrl.Unlock()
	if ctrl.sandbox == nil {
		return
	}

	// reconcile
	ctrl.log.Debug("reconciling tunnels")
	// put the xwls in a map
	xwMap := make(map[string]*tunapiv1.ExternalWorkload, len(ctrl.sandbox.ExternalWorkloads))
	for _, xw := range ctrl.sandbox.ExternalWorkloads {
		xwMap[xw.Name] = xw
	}

	for xwName, xw := range xwMap {
		rt, has := ctrl.revtuns[xwName]
		if has {
			select {
			case <-rt.rtToClose:
				// delete the revtun if it has been closed
				has = false
				delete(ctrl.revtuns, xwName)
			default:
			}
		}

		if has {
			// check connection issues
			if !xw.Connected {
				now := time.Now()
				if rt.clusterNotConnectedTime == nil {
					rt.clusterNotConnectedTime = &now
				}
				if time.Since(*rt.clusterNotConnectedTime) > 10*time.Second {
					// this revtun has been down for more than 10 secs
					// close and delete the current revtune
					has = false
					ctrl.closeRevTunnel(xwName)
					delete(ctrl.revtuns, xwName)
				}
			} else {
				// reset this value
				rt.clusterNotConnectedTime = nil
			}
		}
		if has {
			continue
		}

		// create revtun
		rt, err := newRevtun(ctrl.log, ctrl.revtunClient, ctrl.sandbox.RoutingKey, xw)
		if err != nil {
			ctrl.log.Error("error creating revtun", "error", err)
			continue
		}
		ctrl.revtuns[xwName] = rt
	}

	// delete unwanted tunnels
	for xwName := range ctrl.revtuns {
		_, desired := xwMap[xwName]
		if desired {
			continue
		}
		// close and delete the current revtune
		ctrl.closeRevTunnel(xwName)
		delete(ctrl.revtuns, xwName)
	}
}

func (ctrl *sbController) closeRevTunnel(xwName string) {
	revtun := ctrl.revtuns[xwName]
	if revtun == nil {
		return
	}

	select {
	case <-revtun.rtToClose:
	default:
		close(revtun.rtToClose)
		ctrl.log.Debug("sandbox controller closing revtun", "local", xwName)
	}
}

func (ctrl *sbController) compareExternalWorkloadsSpec(a, b *apiv1.ExternalWorkload) bool {
	// for comparison, ignore status info
	specA := &apiv1.ExternalWorkload{
		Name:                a.Name,
		Baseline:            a.Baseline,
		WorkloadPortMapping: a.WorkloadPortMapping,
	}
	specB := &apiv1.ExternalWorkload{
		Name:                b.Name,
		Baseline:            b.Baseline,
		WorkloadPortMapping: b.WorkloadPortMapping,
	}
	// compare specs
	return proto.Equal(specA, specB)
}
