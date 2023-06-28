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
	routingKey   string
	clapiErrC    chan<- error
	clapiClientC chan clapiclient.Client
	done         chan struct{}
	status       *clapi.WatchSandboxStatus
	xws          map[string]*xwRevtun
}

type xwRevtun struct {
	rtConfig *rtproto.Config
	rtCloser io.Closer
	rtClosed <-chan struct{}
	rtErr    error
}

func newSBMonitor(rk string, clapiErrC chan<- error) *sbMonitor {
	res := &sbMonitor{
		routingKey:   rk,
		clapiErrC:    clapiErrC,
		clapiClientC: make(chan clapiclient.Client, 1),
		done:         make(chan struct{}),
		xws:          make(map[string]*xwRevtun),
	}
	res.monitor()
	return res
}

func (sbm *sbMonitor) monitor() {
	var (
		clapiClient clapiclient.Client
		err         error
		sbwClient   clapi.TunnelAPI_WatchSandboxClient
	)
	// setup context for grp stream requests
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-sbm.done
		cancel()
	}()

	for {
		clapiClient = sbm.clapiClient(clapiClient, err)
		if clapiClient == nil {
			return
		}
		sbwClient, err = clapiClient.WatchSandbox(ctx, &clapi.WatchSandboxRequest{
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

func (sbm *sbMonitor) clapiClient(cc clapiclient.Client, err error) clapiclient.Client {
	if err != nil {
		select {
		case sbm.clapiErrC <- err:
		default:
		}
		select {
		case c := <-sbm.clapiClientC:
			return c
		case <-sbm.done:
			return nil
		}
	}
	if cc == nil {
		select {
		case c := <-sbm.clapiClientC:
			return c
		case <-sbm.done:
			return nil
		}
	}
	return cc
}

func (sbm *sbMonitor) setStatus(st *clapi.WatchSandboxStatus) {
	sbm.Lock()
	defer sbm.Unlock()
	sbm.status = st
}
