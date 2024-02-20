package rootmanager

import (
	"fmt"
	"time"

	"log/slog"

	sbmanagerapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	"github.com/signadot/libconnect/fwdtun/ipmap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	checkingPeriodNotOK = time.Second
	checkingPeriodOK    = 10 * time.Second
)

type pfwMonitor struct {
	log             *slog.Logger
	root            *rootManager
	sbManagerAddr   string
	portforwardAddr string
	ipMap           *ipmap.IPMap
	starting        bool
	sbClient        sbmanagerapi.SandboxManagerAPIClient
	closeCh         chan struct{}
}

func NewPortForwardMonitor(ctx context.Context, root *rootManager, ipMap *ipmap.IPMap) *pfwMonitor {
	mon := &pfwMonitor{
		log:           root.log,
		ipMap:         ipMap,
		sbManagerAddr: fmt.Sprintf("127.0.0.1:%d", root.ciConfig.APIPort),
		root:          root,
		starting:      true,
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
	mon.log.Debug("checking port-forward")
	if mon.sbClient == nil {
		// Establish the connection if needed
		grpcConn, err := grpc.Dial(mon.sbManagerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			if mon.starting {
				mon.log.Debug("waiting for sandbox manager to be ready (not running yet)")
			} else {
				mon.log.Warn("couldn't connect with sandbox manager", "error", err)
			}
			return false
		}
		mon.sbClient = sbmanagerapi.NewSandboxManagerAPIClient(grpcConn)
	}
	mon.log.Debug("connected to sandbox manager")

	// Get the sandbox manager status
	status, err := mon.sbClient.Status(ctx, &sbmanagerapi.StatusRequest{})
	if err != nil {
		if mon.starting {
			mon.log.Debug("waiting for sandbox manager to be ready (api not ready)")
		} else {
			mon.log.Warn("couldn't get status from sandbox manager", "error", err)
		}
		return false
	}

	// Check the status
	if status.Portforward == nil || status.Portforward.Health == nil || !status.Portforward.Health.Healthy {
		mon.log.Debug("port forward not ready in sandbox manager")
		return false
	}
	if status.Portforward.LocalAddress != mon.portforwardAddr {
		mon.portforwardAddr = status.Portforward.LocalAddress
		mon.log.Info("port forward is ready (restarting)", "addr", mon.portforwardAddr, "was", status.Portforward.LocalAddress)

		// Restart localnet
		mon.root.stopLocalnetService()
		mon.root.runLocalnetService(ctx, mon.portforwardAddr, mon.ipMap)

		// Restart etc hosts
		mon.root.stopEtcHostsService()
		mon.root.runEtcHostsService(ctx, mon.portforwardAddr, mon.ipMap)
	}
	mon.starting = false
	return true
}
