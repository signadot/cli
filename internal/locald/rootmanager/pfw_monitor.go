package rootmanager

import (
	"fmt"
	"time"

	sbmanagerapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	"golang.org/x/exp/slog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	checkingPeriodNotOK = time.Second
	checkingPeriodOK    = 10 * time.Second
)

type pfwMonitor struct {
	log            *slog.Logger
	root           *rootManager
	sbManagerAddr  string
	portfowardAddr string
	grpcConn       *grpc.ClientConn
	sbclient       sbmanagerapi.SandboxManagerAPIClient
	closeCh        chan struct{}
}

func NewPortForwardMonitor(ctx context.Context, root *rootManager) *pfwMonitor {
	mon := &pfwMonitor{
		log:           root.log,
		sbManagerAddr: fmt.Sprintf("127.0.0.1:%d", root.conf.APIPort),
		root:          root,
		closeCh:       make(chan struct{}),
	}
	go mon.run(ctx)
	return mon
}

func (mon *pfwMonitor) Stop() {
	select {
	case <-mon.closeCh:
		return
	default:
		close(mon.closeCh)
	}
}

func (mon *pfwMonitor) run(ctx context.Context) {
	ticker := time.NewTicker(checkingPeriodNotOK)
	defer ticker.Stop()

	for {
		if mon.checkPortForward(ctx) {
			ticker.Reset(checkingPeriodOK)
		}
		select {
		case <-ctx.Done():
			// Context is done
			return
		case <-mon.closeCh:
			// We have been stopped
			return
		case <-ticker.C:
			// Check ticker
		}
	}

}

func (mon *pfwMonitor) checkPortForward(ctx context.Context) bool {
	if mon.grpcConn == nil {
		// Establish the connection if needed
		grpcConn, err := grpc.Dial(mon.sbManagerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			mon.log.Warn("couldn't connect with sandbox manager", "error", err)
			return false
		}
		mon.grpcConn = grpcConn
		mon.sbclient = sbmanagerapi.NewSandboxManagerAPIClient(grpcConn)
	}

	// Get the sandbox manager status
	status, err := mon.sbclient.Status(ctx, &sbmanagerapi.StatusRequest{})
	if err != nil {
		mon.log.Warn("couldn't get status from sandbox manager", "error", err)
		return false
	}

	// Check the status
	if status.Portfoward == nil || status.Portfoward.Health == nil || !status.Portfoward.Health.Healthy {
		mon.log.Debug("port forward not ready in sandbox manager")
		return false
	}
	if status.Portfoward.LocalAddress != mon.portfowardAddr {
		mon.portfowardAddr = status.Portfoward.LocalAddress
		mon.log.Info("port forward is ready", "addr", mon.portfowardAddr)

		// Restart localnet
		mon.root.stopLocalnetService()
		mon.root.runLocalnetService(ctx, mon.portfowardAddr)

		// Restart etc hosts
		mon.root.stopEtcHostsService()
		mon.root.runEtcHostsService(ctx, mon.portfowardAddr)
	}
	return true
}
