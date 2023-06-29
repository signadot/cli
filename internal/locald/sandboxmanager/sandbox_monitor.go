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
	res.monitor()
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
		for {
			sbStatus, err := sbwClient.Recv()
			if err == nil {
				sbm.setStatus(sbStatus)
				continue
			}
			var (
				st *status.Status
				ok bool
			)
			if st, ok = status.FromError(err); !ok {
				sbm.log.Error("sandbox monitor grpc stream error: no status",
					"error", err)
				break
			}
			switch st.Code() {
			case codes.OK:
				sbm.setStatus(sbStatus)
				continue
			case codes.Internal:
				sbm.log.Error("sandbox watch: internal grpc error",
					"error", err)

			case codes.NotFound:
				sbm.log.Info("sandbox watch: sandbox not found")
				break
			default:
				sbm.log.Error("sandbox watch error", "error", err)
				break
			}
			sbm.log.Error("sandbox watch client error (non-grpc-status-error)",
				"error", err)
			// TODO deal with
			// rpc error: code = Internal desc = stream terminated by RST_STREAM with error code: NO_ERROR
			break

		}
		// we're done, clean up revtuns and parent delete func
		sbm.delFn()
		sbm.reconcileLocals(nil)
		sbm.setStatus(&clapi.WatchSandboxStatus{})
	}
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
		// TODO create revtun
		rt, err := newXWRevtun(sbm.log.With("local", localName),
			sbm.revtunClient, localName, sbm.routingKey, local)
		if err != nil {
		}
		sbm.revtuns[localName] = rt

	}
}

func (sbm *sbMonitor) setStatus(st *clapi.WatchSandboxStatus) {
	sbm.Lock()
	defer sbm.Unlock()
	sbm.status = st
	// reconcile
	statusXWs := make(map[string]*clapi.WatchSandboxStatus_ExternalWorkload, len(st.ExternalWorkloads))
	// put the xwls in a map
	for _, xw := range st.ExternalWorkloads {
		statusXWs[xw.Name] = xw
	}
	for xwName, sxw := range statusXWs {
		_, has := sbm.revtuns[xwName]
		if has {
			continue
		}
		local := sbm.locals[xwName]
		if local == nil {
			sbm.log.Error("no local found for clust extworkload status", "local", xwName)
			continue
		}
		// create revtun
		rt, err := newXWRevtun(sbm.log.With("local", sxw.Name),
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
			close(xw.rtToClose)
		}
	}
}

func (sbm *sbMonitor) getStatus() *clapi.WatchSandboxStatus {
	sbm.Lock()
	defer sbm.Unlock()
	return sbm.status
}
