package rootmanager

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"log/slog"

	sbmanagerapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	connectcfg "github.com/signadot/libconnect/config"
	"github.com/signadot/libconnect/fwdtun/ipmap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	checkingPeriodNotOK = time.Second
	checkingPeriodOK    = 10 * time.Second
)

type tpMonitor struct {
	log            *slog.Logger
	root           *rootManager
	sbManagerAddr  string
	tpLocalAddr    string
	ipMap          *ipmap.IPMap
	starting       bool
	beginStarting  time.Time
	sbClient       sbmanagerapi.SandboxManagerAPIClient
	closeCh        chan struct{}
	connectTimeout time.Duration
}

func NewTunnelProxyMonitor(ctx context.Context, root *rootManager, ipMap *ipmap.IPMap) *tpMonitor {
	dur, err := time.ParseDuration(root.ciConfig.ConnectTimeout)
	if err != nil {
		// error should not occur b/c it is set and processed on the calling local connect
		// with default 10 seconds
		dur = 10 * time.Second
	}
	mon := &tpMonitor{
		log:            root.log,
		ipMap:          ipMap,
		sbManagerAddr:  fmt.Sprintf("127.0.0.1:%d", root.ciConfig.APIPort),
		root:           root,
		starting:       true,
		beginStarting:  time.Now(),
		closeCh:        make(chan struct{}),
		connectTimeout: dur,
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

	mon.starting = true
	mon.beginStarting = time.Now()
	for {
		success, restarted := mon.checkTunnelProxyAccess(ctx)
		if success {
			ticker.Reset(checkingPeriodOK)
			mon.starting = false
		} else {
			ticker.Reset(checkingPeriodNotOK)
			if restarted || !mon.starting {
				mon.beginStarting = time.Now()
			}
			mon.starting = true
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

func (mon *tpMonitor) checkTunnelProxyAccess(ctx context.Context) (success bool, restarted bool) {
	mon.log.Debug("checking tunnel-proxy access")
	if mon.sbClient == nil {
		// Establish the connection if needed
		grpcConn, err := grpc.NewClient(mon.sbManagerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			if mon.starting {
				mon.log.Debug("waiting for sandbox manager to be ready (not running yet)")
			} else {
				mon.log.Warn("couldn't connect with sandbox manager", "error", err)
			}
			return false, false
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
		return false, false
	}

	// Check the status
	ok, restart := mon.checkLinkStatus(status)
	if !ok {
		return false, restart
	}
	if !restart {
		ok, restart = mon.checkRootServer(mon.root.root)
		if !ok {
			return false, restart
		}
	}
	if !restart {
		// things are looking ok but the grpc check for connecting to the tunnel proxy does not suffice
		// because it has built-in retries and may re-use a connection while
		// we are unable to establish a new connection.  So, we also check
		// the agent-metrics endpoint.
		cli := &http.Client{
			// don't cache connections, use fresh transport
			Transport: &http.Transport{},
			Timeout:   10 * time.Second,
		}
		resp, err := cli.Get("http://agent-metrics.signadot.svc:9090/metrics")
		if err != nil {
			if mon.shouldRestartDueToUnhealthy() {
				mon.log.Error("unable to reach agent-metrics, restarting services", "error", err)
				restart = true
			} else {
				mon.log.Debug("unable to reach agent-metrics, but still in startup grace period", "error", err)
				return false, false
			}
		} else {
			resp.Body.Close()
		}
	}
	if !restart {
		return true, false
	}

	mon.log.Info("restarting localnet and etchosts services")

	// Restart localnet
	mon.root.stopLocalnetService()
	mon.root.runLocalnetService(ctx, mon.tpLocalAddr, mon.ipMap)

	// Restart etc hosts
	mon.root.stopEtcHostsService()
	mon.root.runEtcHostsService(ctx, mon.tpLocalAddr, mon.ipMap)

	return false, true
}

func (mon *tpMonitor) checkLinkStatus(status *sbmanagerapi.StatusResponse) (ok, restart bool) {
	switch mon.root.ciConfig.ConnectionConfig.Type {
	case connectcfg.PortForwardLinkType:
		if status.Portforward == nil || status.Portforward.Health == nil ||
			!status.Portforward.Health.Healthy {
			mon.log.Debug("port-forward not ready in sandbox manager")
			return false, false
		}
		if status.Portforward.LocalAddress != mon.tpLocalAddr {
			mon.log.Info("port forward is ready", "addr", status.Portforward.LocalAddress, "was", mon.tpLocalAddr)
			mon.tpLocalAddr = status.Portforward.LocalAddress
			restart = true
		}
	case connectcfg.ControlPlaneProxyLinkType:
		if status.ControlPlaneProxy == nil || status.ControlPlaneProxy.Health == nil ||
			!status.ControlPlaneProxy.Health.Healthy {
			mon.log.Debug("control-plane proxy not ready in sandbox manager")
			return false, false
		}
		if status.ControlPlaneProxy.LocalAddress != mon.tpLocalAddr {
			mon.log.Info("control-plane proxy is ready", "addr", status.ControlPlaneProxy.LocalAddress, "was", mon.tpLocalAddr)
			mon.tpLocalAddr = status.ControlPlaneProxy.LocalAddress
			restart = true
		}
	}
	return true, restart
}

func (mon *tpMonitor) checkRootServer(r *rootServer) (ok, restart bool) {
	if r.localnetSVC == nil || !r.localnetSVC.Status().Healthy {
		if mon.shouldRestartDueToUnhealthy() {
			return true, true
		} else {
			return false, false
		}
	}
	// localnet ok, check etc hosts
	if r.etcHostsSVC == nil || !r.etcHostsSVC.Status().Healthy {
		if mon.shouldRestartDueToUnhealthy() {
			return true, true
		} else {
			return false, false
		}
	}
	return true, false
}

// shouldRestartDueToUnhealthy determines whether services should be restarted
// when they become unhealthy. During the initial startup phase (first 10 seconds),
// it waits to give services time to become healthy. After startup, it immediately
// indicates that services should be restarted.
func (mon *tpMonitor) shouldRestartDueToUnhealthy() bool {
	if mon.starting {
		return time.Since(mon.beginStarting) > mon.connectTimeout
	}
	return true
}
