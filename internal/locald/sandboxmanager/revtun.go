package sandboxmanager

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/signadot/go-sdk/models"
	"github.com/signadot/libconnect/revtun"
	rtproto "github.com/signadot/libconnect/revtun/protocol"
	"golang.org/x/exp/slog"
)

type rt struct {
	log       *slog.Logger
	localName string
	rtClient  revtun.Client
	rtConfig  *rtproto.Config
	rtCloser  io.Closer
	rtClosed  <-chan struct{}
	rtToClose chan struct{}
	rtErr     error
}

func newXWRevtun(log *slog.Logger, rtc revtun.Client, name, rk string, local *models.Local) (*rt, error) {
	rtConfig := &rtproto.Config{
		SandboxRoutingKey: rk,
		ExternalWorkload:  name,
		Forwards:          []rtproto.Forward{},
	}
	for _, pm := range local.Mappings {
		kind, err := kindToRemoteURLTLD(local.From.Kind)
		if err != nil {
			return nil, err
		}
		rtConfig.Forwards = append(rtConfig.Forwards,
			rtproto.Forward{
				LocalURL: fmt.Sprintf("tcp://%s", pm.ToLocal),
				RemoteURL: fmt.Sprintf("tcp://%s.%s.%s:%d",
					local.From.Name,
					local.From.Namespace,
					kind,
					pm.Port),
			},
		)
	}
	res := &rt{
		log:       log.With("sandbox", rk, "local", name),
		localName: name,
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
			<-time.After(time.Second)
			continue
		}
		t.log.Info("reverse tunnel is setup")
		select {
		case <-t.rtClosed:
			t.log.Info("closed, retrying")
		case <-t.rtToClose:
			t.rtCloser.Close()
			return
		}
	}
}

func kindToRemoteURLTLD(kind *string) (string, error) {
	if kind == nil {
		return "svc", nil
	}
	switch *kind {
	case "Deploy":
		return "deploy", nil
	case "Rollout":
		return "rollout", nil
	default:
		return "", fmt.Errorf("invalid local.From kind: %q", kind)
	}
}
