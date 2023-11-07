package sandboxmanager

import (
	"context"
	"fmt"
	"io"
	"time"

	"log/slog"

	tunapiv1 "github.com/signadot/libconnect/apiv1"
	"github.com/signadot/libconnect/revtun"
	rtproto "github.com/signadot/libconnect/revtun/protocol"
)

type rt struct {
	log *slog.Logger

	// NB used exclusively by sanbox controller => no races
	clusterNotConnectedTime *time.Time
	rtClient                revtun.Client
	rtConfig                *rtproto.Config
	rtCloser                io.Closer
	rtClosed                <-chan struct{}
	rtToClose               chan struct{}
	rtErr                   error
}

func newRevtun(log *slog.Logger, rtc revtun.Client, rk string,
	xw *tunapiv1.WatchLocalSandboxesResponse_ExternalWorkload) (*rt, error) {
	// define the revtun config (that will be used to setup the reverse tunnel)
	rtConfig := &rtproto.Config{
		SandboxRoutingKey: rk,
		ExternalWorkload:  xw.Name,
		Forwards:          []rtproto.Forward{},
	}
	for _, pm := range xw.WorkloadPortMapping {
		kind, err := kindToRemoteURLTLD(xw.Baseline.Kind)
		if err != nil {
			return nil, err
		}
		rtConfig.Forwards = append(rtConfig.Forwards,
			rtproto.Forward{
				LocalURL: fmt.Sprintf("tcp://%s", pm.LocalAddress),
				RemoteURL: fmt.Sprintf("tcp://%s.%s.%s:%d",
					xw.Baseline.Name,
					xw.Baseline.Namespace,
					kind,
					pm.BaselinePort,
				),
			},
		)
	}
	res := &rt{
		log:       log.With("local", xw.Name),
		rtClient:  rtc,
		rtConfig:  rtConfig,
		rtToClose: make(chan struct{}),
	}
	go res.monitor()
	return res, nil
}

func (t *rt) monitor() {
	for {
		setupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		func() {
			defer cancel()
			t.rtCloser, t.rtClosed, t.rtErr = t.rtClient.Setup(setupCtx, t.rtConfig)
		}()
		if t.rtErr != nil {
			t.log.Error("error setting up revtun", "error", t.rtErr)
			// wait until closed or retry in 1 sec
			select {
			case <-t.rtToClose:
				return
			case <-time.After(time.Second):
				continue
			}
		}
		// wait until closed and reconnect if tunnel goes down
		t.log.Info("reverse tunnel is setup", "config", t.rtConfig.Key())
		select {
		case <-t.rtClosed:
			t.log.Info("closed, retrying")
		case <-t.rtToClose:
			t.log.Debug("closing reverse tunnel", "config", t.rtConfig.Key())
			t.rtCloser.Close()
			return
		}
	}
}

func kindToRemoteURLTLD(kind string) (string, error) {
	switch kind {
	case "Deployment":
		return "deploy", nil
	case "Rollout":
		return "rollout", nil
	default:
		return "", fmt.Errorf("invalid baseline kind: %q", kind)
	}
}
