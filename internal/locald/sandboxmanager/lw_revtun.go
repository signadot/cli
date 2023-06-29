package sandboxmanager

import (
	"io"

	"github.com/signadot/libconnect/revtun"
	rtproto "github.com/signadot/libconnect/revtun/protocol"
	"golang.org/x/exp/slog"
)

type xwRevtun struct {
	log      *slog.Logger
	rtConfig *rtproto.Config
	rtCloser io.Closer
	rtClosed <-chan struct{}
	rtErr    error
}

func newXWRevtun(log *slog.Logger, name, user, rk string) (*xwRevtun, error) {
	rtConfig := &rtproto.Config{
		SandboxRoutingKey: rk,
		ExternalWorkload:  name,
		User:              user,
		Forwards:          []rtproto.Forward{},
	}
	_ = rtConfig
	rtClientConfig := &revtun.ClientConfig{
		User: user,
	}
	_ = rtClientConfig
	return nil, nil
}
