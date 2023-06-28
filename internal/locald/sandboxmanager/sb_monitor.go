package sandboxmanager

import (
	"context"
	"io"
	"sync"

	clapi "github.com/signadot/libconnect/apiv1"
	clapiclient "github.com/signadot/libconnect/common/apiclient"
	rtproto "github.com/signadot/libconnect/revtun/protocol"
)

type sbMonitor struct {
	sync.Mutex
	routingKey  string
	clapiClient clapiclient.Client
	done        chan struct{}
	status      *clapi.WatchSandboxStatus
	xws         map[string]*xwRevtun
}

type xwRevtun struct {
	rtConfig *rtproto.Config
	rtCloser io.Closer
	rtClosed <-chan struct{}
	rtErr    error
}

func newSBMonitor(rk string, clapiClient clapiclient.Client) *sbMonitor {
	res := &sbMonitor{
		routingKey:  rk,
		clapiClient: clapiClient,
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
			continue
		}
		for {
			sbst, err := sbwClient.Recv()
			if err != nil {
				// TODO sandbox gone, detect
				// client errors
				break
			}
			sbm.setStatus(sbst)
		}
	}
}

func (sbm *sbMonitor) setStatus(st *clapi.WatchSandboxStatus) {
	sbm.Lock()
	defer sbm.Unlock()
	sbm.status = st
}
