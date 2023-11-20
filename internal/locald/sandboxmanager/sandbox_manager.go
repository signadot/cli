package sandboxmanager

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"log/slog"

	"github.com/signadot/cli/internal/config"
	sbapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	"github.com/signadot/cli/internal/utils/system"
	tunapiclient "github.com/signadot/libconnect/common/apiclient"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"

	// load all auth plugins
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/signadot/libconnect/common/portforward"
	connectcfg "github.com/signadot/libconnect/config"
	"github.com/signadot/libconnect/revtun"
	"github.com/signadot/libconnect/revtun/protocol"
	"github.com/signadot/libconnect/revtun/sshrevtun"
	"github.com/signadot/libconnect/revtun/xaprevtun"
)

type sandboxManager struct {
	log         *slog.Logger
	ciConfig    *config.ConnectInvocationConfig
	connConfig  *connectcfg.ConnectionConfig
	hostname    string
	machineID   string
	grpcServer  *grpc.Server
	portForward *portforward.PortForward
	shutdownCh  chan struct{}

	// tunnel API
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

	// Resolve the machine ID
	machineID, err := system.GetMachineID()
	if err != nil {
		return nil, err
	}

	ciConfig := cfg.ConnectInvocationConfig

	return &sandboxManager{
		log:        log,
		ciConfig:   ciConfig,
		connConfig: ciConfig.ConnectionConfig,
		hostname:   hostname,
		machineID:  machineID,
		grpcServer: grpcServer,
		shutdownCh: shutdownCh,
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

	// Create an operator info updater
	oiu := &operatorInfoUpdater{
		log: m.log,
	}

	// Create the watcher
	sbmWatcher := newSandboxManagerWatcher(m.log, m.machineID, m.revtunClient, oiu, m.shutdownCh)

	// Register our service in gRPC server
	sbmServer := newSandboxManagerGRPCServer(m.log, m.ciConfig, m.portForward,
		sbmWatcher, oiu, m.shutdownCh)
	sbapi.RegisterSandboxManagerAPIServer(m.grpcServer, sbmServer)

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

	// Run the sandboxes watcher
	sbmWatcher.run(runCtx, m.tunAPIClient)

	// Wait until termination
	<-runCtx.Done()

	// Clean up
	m.log.Info("Shutting down")
	m.grpcServer.GracefulStop()
	sbmWatcher.stop()
	if m.portForward != nil {
		m.portForward.Close()
	}
	return nil
}

func (m *sandboxManager) runAPIServer() error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", m.ciConfig.APIPort))
	if err != nil {
		return err
	}
	go m.grpcServer.Serve(ln)
	return nil
}

func (m *sandboxManager) setTunnelAPIClient(proxyAddress string) error {
	tunAPIClient, err := tunapiclient.NewClient(proxyAddress, tunapiclient.DefaultClientKeepaliveParams())
	if err != nil {
		return err
	}

	m.tunAPIClient = tunAPIClient
	m.proxyAddress = proxyAddress
	return nil
}

func (m *sandboxManager) revtunClient() revtun.Client {
	rtClientConfig := &revtun.ClientConfig{
		Labels: map[string]string{
			protocol.RevTunnelUserLabel:     m.ciConfig.User.Username,
			protocol.RevTunnelHostnameLabel: m.hostname,
		},
		Socks5Addr: m.proxyAddress,
		ErrFunc: func(e error) {
			m.log.Error("revtun.errfunc", "error", e)
		},
	}
	inboundProto := connectcfg.SSHInboundProtocol
	if m.connConfig.Inbound != nil {
		inboundProto = m.connConfig.Inbound.Protocol
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
		panic(fmt.Errorf("invalid inbound protocol: %s", m.connConfig.Inbound.Protocol))
	}
}
