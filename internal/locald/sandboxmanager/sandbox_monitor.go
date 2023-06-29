package sandboxmanager

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/signadot/go-sdk/models"
	clapi "github.com/signadot/libconnect/apiv1"
	clapiclient "github.com/signadot/libconnect/common/apiclient"
	"github.com/signadot/libconnect/revtun"
	"golang.org/x/exp/slog"
)

type sbMonitor struct {
	sync.Mutex
	routingKey  string
	clapiClient clapiclient.Client
	// func called on delete
	delFn            func()
	log              *slog.Logger
	done             chan struct{}
	status           *clapi.WatchSandboxStatus
	revtuns          map[string]*rt
	locals           map[string]*models.Local
	revtunClientFunc func() revtun.Client
}

func newSBMonitor(rk string, clapiClient clapiclient.Client, rtClientFunc func() revtun.Client, delFn func(), log *slog.Logger) *sbMonitor {
	res := &sbMonitor{
		routingKey:       rk,
		clapiClient:      clapiClient,
		revtunClientFunc: rtClientFunc,
		delFn:            delFn,
		log:              log,
		done:             make(chan struct{}),
		locals:           make(map[string]*models.Local),
		revtuns:          make(map[string]*rt),
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
			sbm.log.Error("error getting sb watch stream", "error", err)
			continue
		}
		for {
			sbStatus, err := sbwClient.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					sbm.log.Debug("sandbox monitor eof",
						"routing-key", sbm.routingKey)
					break
				}
				sbm.log.Error("sandbox monitor grpc stream error",
					"routing-key", sbm.routingKey,
					"error", err)
				break
			}
			sbm.setStatus(sbStatus)
		}
		// we're done
		sbm.delFn()
		sbm.reconcileLocals(nil)
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
		sbm.revtuns[localName] = nil
	}
}

func (sbm *sbMonitor) setStatus(st *clapi.WatchSandboxStatus) {
	sbm.Lock()
	defer sbm.Unlock()
	sbm.status = st
	// reconcile
	statusXWs := make(map[string]*clapi.WatchSandboxStatus_ExternalWorkload, len(st.ExternalWorkloads))
	for _, xw := range st.ExternalWorkloads {
		statusXWs[xw.Name] = xw
	}
	for xwName, sxw := range statusXWs {
		_, has := sbm.revtuns[xwName]
		if has {
			continue
		}
		// create revtun
		sbm.revtuns[xwName] = nil
		_ = sxw
	}
	for xwName, xw := range sbm.revtuns {
		_, desired := statusXWs[xwName]
		if !desired {
			xw.rtCloser.Close()
			delete(sbm.revtuns, xwName)
			continue
		}
	}
}

func (sbm *sbMonitor) getStatus() *clapi.WatchSandboxStatus {
	sbm.Lock()
	defer sbm.Unlock()
	return sbm.status
}
