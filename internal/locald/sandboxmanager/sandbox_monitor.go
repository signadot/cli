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

type sbMonitor struct {
	sync.Mutex
	routingKey   string
	clapiClient  clapiclient.Client
	revtunClient revtun.Client
	// func called on delete
	delFn   func()
	log     *slog.Logger
	done    chan struct{}
	status  *clapi.WatchSandboxStatus
	revtuns map[string]*rt
	locals  map[string]*models.Local
}

func newSBMonitor(rk string, clapiClient clapiclient.Client, rtClient revtun.Client, delFn func(), log *slog.Logger) *sbMonitor {
	res := &sbMonitor{
		routingKey:   rk,
		clapiClient:  clapiClient,
		revtunClient: rtClient,
		delFn:        delFn,
		log:          log,
		done:         make(chan struct{}),
		locals:       make(map[string]*models.Local),
		revtuns:      make(map[string]*rt),
	}
	go res.monitor()
	return res
}

func (sbm *sbMonitor) monitor() {
	var (
		err       error
		sbwClient clapi.TunnelAPI_WatchSandboxClient
	)
	// setup context for grp stream requests
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-sbm.done
		cancel()
	}()

	for {
		sbwClient, err = sbm.clapiClient.WatchSandbox(ctx, &clapi.WatchSandboxRequest{
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
	sbm.log.Debug("cleaning up status and locals and parent")
	// we're done, clean up revtuns and parent delete func
	sbm.reconcileLocals(nil)
	sbm.reconcileStatus(&clapi.WatchSandboxStatus{})
	sbm.delFn()
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
			sbm.reconcileStatus(sbStatus)
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
			sbm.reconcileStatus(sbStatus)
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
		// TODO deal with
		// rpc error: code = Internal desc = stream terminated by RST_STREAM with error code: NO_ERROR
		break

	}
	return err
}

func (sbm *sbMonitor) reconcileLocals(locals []*models.Local) {
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
		rt, err := newRevtun(sbm.log.With("local", localName),
			sbm.revtunClient, localName, sbm.routingKey, local)
		if err != nil {
			panic(err)
		}
		sbm.revtuns[localName] = rt
	}
}

func (sbm *sbMonitor) reconcileStatus(st *clapi.WatchSandboxStatus) {
	sbm.Lock()
	defer sbm.Unlock()
	sbm.status = st
	// reconcile
	statusXWs := make(map[string]*clapi.WatchSandboxStatus_ExternalWorkload, len(st.ExternalWorkloads))
	// put the xwls in a map
	for _, xw := range st.ExternalWorkloads {
		statusXWs[xw.Name] = xw
	}
	sbm.log.Debug("sbm setting watch status", "status", st)
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
			}
		}
		if has {
			continue
		}
		local := sbm.locals[xwName]
		if local == nil {
			sbm.log.Error("no local found for clust extworkload status", "local", xwName)
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
		}
	}
}

func (sbm *sbMonitor) getStatus() *clapi.WatchSandboxStatus {
	sbm.Lock()
	defer sbm.Unlock()
	return sbm.status
}
