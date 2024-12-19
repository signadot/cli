package rootmanager

import (
	"fmt"
	"net/http"
	"time"

	"log/slog"

	sbmanagerapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	connectcfg "github.com/signadot/libconnect/config"
	"github.com/signadot/libconnect/fwdtun/ipmap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	checkingPeriodNotOK = time.Second
	checkingPeriodOK    = 10 * time.Second
)

type tpMonitor struct {
	log           *slog.Logger
	root          *rootManager
	sbManagerAddr string
	tpLocalAddr   string
	ipMap         *ipmap.IPMap
	starting      bool
	sbClient      sbmanagerapi.SandboxManagerAPIClient
	closeCh       chan struct{}
}

func NewTunnelProxyMonitor(ctx context.Context, root *rootManager, ipMap *ipmap.IPMap) *tpMonitor {
	mon := &tpMonitor{
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

func (mon *tpMonitor) Stop() {
	select {
	case <-mon.closeCh:
		return
	default:
		close(mon.closeCh)
	}
}

func (mon *tpMonitor) run(ctx context.Context) {
	ticker := time.NewTicker(checkingPeriodNotOK)
	defer ticker.Stop()

	for {
		if mon.checkTunnelProxyAccess(ctx) {
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

func (mon *tpMonitor) checkTunnelProxyAccess(ctx context.Context) bool {
	mon.log.Debug("checking tunnel-proxy access")
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
	restartSvcs := false
	switch mon.root.ciConfig.ConnectionConfig.Type {
	case connectcfg.PortForwardLinkType:
		if status.Portforward == nil || status.Portforward.Health == nil ||
			!status.Portforward.Health.Healthy {
			mon.log.Debug("port-forward not ready in sandbox manager")
			return false
		}
		if status.Portforward.LocalAddress != mon.tpLocalAddr {
			mon.log.Info("port forward is ready", "addr", status.Portforward.LocalAddress, "was", mon.tpLocalAddr)
			mon.tpLocalAddr = status.Portforward.LocalAddress
			restartSvcs = true
		}
	case connectcfg.ControlPlaneProxyLinkType:
		if status.ControlPlaneProxy == nil || status.ControlPlaneProxy.Health == nil ||
			!status.ControlPlaneProxy.Health.Healthy {
			mon.log.Debug("control-plane proxy not ready in sandbox manager")
			return false
		}
		if status.ControlPlaneProxy.LocalAddress != mon.tpLocalAddr {
			mon.log.Info("control-plane proxy is ready", "addr", status.ControlPlaneProxy.LocalAddress, "was", mon.tpLocalAddr)
			mon.tpLocalAddr = status.ControlPlaneProxy.LocalAddress
			restartSvcs = true
		}
	}
	if !restartSvcs {
		if mon.root == nil || mon.root.root == nil || mon.root.root.etcHostsSVC == nil || !mon.root.root.etcHostsSVC.Status().Healthy {
			return false
		}
	}
	if !restartSvcs {
		// the grpc check for connecting to the tunnel proxy does not suffice
		// because it has built-in retries and may re-use a connection while
		// we are unable to establish a new connection.  So, we also check
		// the agent-metrics endpoint.
		cli := &http.Client{
			Transport: &http.Transport{},
			Timeout:   10 * time.Second,
		}
		resp, err := cli.Get("http://agent-metrics.signadot.svc:9090/metrics")
		if err != nil {
			mon.log.Error("unable to reach agent-metrics, restarting services", "error", err)
			//fmt.Printf("unable to reach agent-metrics: %v", err)
			restartSvcs = true
		} else {
			resp.Body.Close()
		}
	}
	if !restartSvcs {
		mon.starting = false
		return true
	}

	// Restart localnet
	mon.root.stopLocalnetService()
	mon.root.runLocalnetService(ctx, mon.tpLocalAddr, mon.ipMap)

	// Restart etc hosts
	mon.root.stopEtcHostsService()
	mon.root.runEtcHostsService(ctx, mon.tpLocalAddr, mon.ipMap)
	return false
}
