package sandboxmanager

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/signadot/cli/internal/config"
	sbapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	tunapiclient "github.com/signadot/libconnect/common/apiclient"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"

	"github.com/signadot/libconnect/common/portforward"
	connectcfg "github.com/signadot/libconnect/config"
	"github.com/signadot/libconnect/revtun"
	"github.com/signadot/libconnect/revtun/protocol"
	"github.com/signadot/libconnect/revtun/sshrevtun"
	"github.com/signadot/libconnect/revtun/xaprevtun"
)

type sandboxManager struct {
	log          *slog.Logger
	localdConfig *config.LocalDaemon
	connConfig   *connectcfg.ConnectionConfig
	apiPort      uint16
	hostname     string
	grpcServer   *grpc.Server
	sbmServer    *sbmServer
	portForward  *portforward.PortForward
	shutdownCh   chan struct{}

	// tunnel API
	tunMu        sync.Mutex
	tunAPIClient tunapiclient.Client
	proxyAddress string
}

func NewSandboxManager(cfg *config.LocalDaemon, args []string, log *slog.Logger) (*sandboxManager, error) {
	shutdownCh := make(chan struct{})
	grpcServer := grpc.NewServer()

	// Resolve the hostname
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return &sandboxManager{
		log:          log,
		localdConfig: cfg,
		connConfig:   cfg.ConnectInvocationConfig.ConnectionConfig,
		apiPort:      cfg.ConnectInvocationConfig.APIPort,
		hostname:     hostname,
		grpcServer:   grpcServer,
		shutdownCh:   shutdownCh,
	}, nil
}

func (m *sandboxManager) Run(ctx context.Context) error {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	runCtx, cancel := context.WithCancel(ctx)
	go func() {
		select {
		case <-runCtx.Done():
		case <-sigs:
			cancel()
		case <-m.shutdownCh:
			cancel()
		}
	}()

	// Start the port-forward (if needed)
	if m.connConfig.Type == connectcfg.PortForwardLinkType {
		restConfig, err := m.connConfig.GetRESTConfig()
		if err != nil {
			return err
		}
		clientSet, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return err
		}
		m.portForward = portforward.NewPortForward(
			runCtx,
			restConfig, clientSet,
			m.log, 0, "signadot", "tunnel-proxy", 1080,
		)
	} else {
		// Create the tunnel API client
		if err := m.setTunnelAPIClient(m.connConfig.ProxyAddress); err != nil {
			return err
		}
	}

	// Register our service in gRPC server
	m.sbmServer = newSandboxManagerGRPCServer(m.localdConfig.ConnectInvocationConfig,
		m.portForward, m.isSBManagerReady, m.getSBMonitor, m.log, m.shutdownCh)
	sbapi.RegisterSandboxManagerAPIServer(m.grpcServer, m.sbmServer)

	// Run the gRPC server
	if err := m.runAPIServer(); err != nil {
		return err
	}

	if m.portForward != nil {
		// Wait until port-forward is healthy
		status, err := m.portForward.WaitHealthy(runCtx)
		if err != nil || status.LocalPort == nil {
			m.log.Error("couldn't establish port-forward", "error", err)
			cancel()
		} else {
			// Create the tunnel API client
			proxyAddress := fmt.Sprintf("localhost:%d", *status.LocalPort)
			m.log.Info("port-forward is connected", "local-addr", proxyAddress)

			if err := m.setTunnelAPIClient(proxyAddress); err != nil {
				m.log.Error("couldn't create tunnel-api client", "error", err)
				cancel()
			}
		}
	}

	// Wait until termination
	<-runCtx.Done()

	// Clean up
	m.log.Info("Shutting down")
	m.grpcServer.GracefulStop()
	m.sbmServer.stop()
	if m.portForward != nil {
		m.portForward.Close()
	}
	return nil
}

func (m *sandboxManager) runAPIServer() error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", m.apiPort))
	if err != nil {
		return err
	}
	go m.grpcServer.Serve(ln)
	return nil
}

func (m *sandboxManager) isSBManagerReady() bool {
	m.tunMu.Lock()
	defer m.tunMu.Unlock()
	return m.tunAPIClient != nil
}

func (m *sandboxManager) getSBMonitor(routingKey string, delFn func()) *sbMonitor {
	m.tunMu.Lock()
	defer m.tunMu.Unlock()
	if m.tunAPIClient == nil {
		// this shouldn't happen because we check
		// if sb manager is ready, which is
		// monotonic
		m.log.Error("invalid internal state: getSBMonitor while sb manager is not ready")
		return nil
	}
	return newSBMonitor(
		routingKey,
		m.tunAPIClient,
		m.revtunClient(),
		delFn,
		m.log.With("sandbox-routing-key", routingKey),
	)
}

func (m *sandboxManager) setTunnelAPIClient(proxyAddress string) error {
	tunAPIClient, err := tunapiclient.NewClient(proxyAddress, tunapiclient.DefaultClientKeepaliveParams())
	if err != nil {
		return err
	}

	m.tunMu.Lock()
	defer m.tunMu.Unlock()
	m.tunAPIClient = tunAPIClient
	m.proxyAddress = proxyAddress
	return nil
}

func (m *sandboxManager) revtunClient() revtun.Client {
	connConfig := m.localdConfig.ConnectInvocationConfig.ConnectionConfig
	rtClientConfig := &revtun.ClientConfig{
		Labels: map[string]string{
			protocol.RevTunnelUserLabel:     connConfig.KubeContext,
			protocol.RevTunnelHostnameLabel: m.hostname,
		},
		Socks5Addr: m.proxyAddress,
		ErrFunc: func(e error) {
			m.log.Error("revtun.errfunc", "error", e)
		},
	}
	inboundProto := connectcfg.SSHInboundProtocol
	if connConfig.Inbound != nil {
		inboundProto = connConfig.Inbound.Protocol
	}
	switch inboundProto {
	case connectcfg.XAPInboundProtocol:
		rtClientConfig.Addr = "localhost:7777"
		return xaprevtun.NewClient(rtClientConfig, "")
	case connectcfg.SSHInboundProtocol:
		rtClientConfig.Addr = "localhost:2222"
		return sshrevtun.NewClient(rtClientConfig, nil)
	default:
		// already validated
		panic(fmt.Errorf("invalid inbound protocol: %s", connConfig.Inbound.Protocol))
	}
}
