package sandboxmanager

import (
	"context"
	"errors"
	"io"
	"sync"

	clapi "github.com/signadot/libconnect/apiv1"
	clapiclient "github.com/signadot/libconnect/common/apiclient"
	"golang.org/x/exp/slog"
)

type sbMonitor struct {
	sync.Mutex
	routingKey  string
	clapiClient clapiclient.Client
	// func called on delete
	delFn  func()
	log    *slog.Logger
	done   chan struct{}
	status *clapi.WatchSandboxStatus
	xws    map[string]*xwRevtun
}

func newSBMonitor(rk string, clapiClient clapiclient.Client, delFn func(), log *slog.Logger) *sbMonitor {
	res := &sbMonitor{
		routingKey:  rk,
		clapiClient: clapiClient,
		delFn:       delFn,
		log:         log,
		done:        make(chan struct{}),
		xws:         make(map[string]*xwRevtun),
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
					sbm.log.Debug("sandbox monitor eof", "routing-key", sbm.routingKey)
					sbm.delFn()
					break
				}
				// TODO sandbox gone, detect
				// client errors
				break
			}
			sbm.setStatus(sbStatus)
		}
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
		_, has := sbm.xws[xwName]
		if has {
			continue
		}
		// create revtun
		sbm.xws[xwName] = nil
		_ = sxw
	}
	for xwName, xw := range sbm.xws {
		_, desired := statusXWs[xwName]
		if !desired {
			xw.rtCloser.Close()
			delete(sbm.xws, xwName)
			continue
		}
	}
}

func (sbm *sbMonitor) getStatus() *clapi.WatchSandboxStatus {
	sbm.Lock()
	defer sbm.Unlock()
	return sbm.status
}
