package sandboxmanager

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log/slog"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/devbox"
	"github.com/signadot/cli/internal/locald/sandboxmanager/apiclient"
	rootapi "github.com/signadot/cli/internal/locald/api/rootmanager"
	sbapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	tunapiclient "github.com/signadot/libconnect/common/apiclient"
	"github.com/signadot/libconnect/common/controlplaneproxy"
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
	log           *slog.Logger
	ciConfig      *config.ConnectInvocationConfig
	connConfig    *connectcfg.ConnectionConfig
	hostname      string
	grpcServer    *grpc.Server
	portForward   *portforward.PortForward
	ctlPlaneProxy *controlplaneproxy.Proxy
	shutdownCh    chan struct{}

	// tunnel API
	tunAPIClient tunapiclient.Client
	proxyAddress string

	// devbox session management
	devboxSessionMgr *devbox.SessionManager

	// session released state (deadend mode)
	sessionReleased bool
	sbmServer       *sbmServer
}

func NewSandboxManager(cfg *config.LocalDaemon, args []string, log *slog.Logger) (*sandboxManager, error) {
	shutdownCh := make(chan struct{})
	grpcServer := grpc.NewServer()

	// Resolve the hostname
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	ciConfig := cfg.ConnectInvocationConfig

	// Create devbox session manager
	devboxSessionMgr, err := devbox.NewSessionManager(log, ciConfig, shutdownCh)
	if err != nil {
		return nil, fmt.Errorf("failed to create devbox session manager: %w", err)
	}

	return &sandboxManager{
		log:              log,
		ciConfig:         ciConfig,
		connConfig:       ciConfig.ConnectionConfig,
		hostname:         hostname,
		grpcServer:       grpcServer,
		shutdownCh:       shutdownCh,
		devboxSessionMgr: devboxSessionMgr,
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

	switch m.connConfig.Type {
	case connectcfg.PortForwardLinkType:
		// Start the port-forward
		restConfig, err := m.connConfig.GetRESTConfig()
		if err != nil {
			return fmt.Errorf("error getting RESTConfig for port-forward: %w", err)
		}
		clientSet, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return fmt.Errorf("error getting k8x clientset for port-forward: %w", err)
		}
		m.portForward = portforward.NewPortForward(
			runCtx,
			restConfig, clientSet,
			m.log, 0, "signadot", "tunnel-proxy", 1080,
		)
	case connectcfg.ControlPlaneProxyLinkType:
		// Use unified API client mechanism for getting auth headers
		getHeaders := func() (http.Header, error) {
			return apiclient.GetAuthHeaders(m.ciConfig)
		}

		// Start a control-plane proxy
		ctlPlaneProxy, err := controlplaneproxy.NewProxy(&controlplaneproxy.Config{
			Log:              m.log,
			ProxyURL:         m.ciConfig.ProxyURL,
			TargetURL:        "tcp://tunnel-proxy.signadot.svc:1080",
			Cluster:          m.connConfig.Cluster,
			BindAddr:         ":0",
			GetInjectHeaders: getHeaders,
		})
		if err != nil {
			return fmt.Errorf("error creating control plane proxy: %w", err)
		}
		m.ctlPlaneProxy = ctlPlaneProxy
		go m.ctlPlaneProxy.Run(ctx)
	default:
		// Create the tunnel API client
		if err := m.setTunnelAPIClient(m.connConfig.ProxyAddress); err != nil {
			return fmt.Errorf("error creating tunnel api client: %w", err)
		}
	}

	// Create an operator info updater
	oiu := &operatorInfoUpdater{
		log: m.log,
	}

	// Create the watcher
	sbmWatcher := newSandboxManagerWatcher(m.log, m.ciConfig.DevboxSessionID, m.revtunClient, oiu, m.shutdownCh)

	// Register our service in gRPC server
	m.sbmServer = newSandboxManagerGRPCServer(m.log, m.ciConfig, m.portForward, m.ctlPlaneProxy,
		sbmWatcher, oiu, m.shutdownCh, m.devboxSessionMgr)
	sbapi.RegisterSandboxManagerAPIServer(m.grpcServer, m.sbmServer)

	// Run the gRPC server
	if err := m.runAPIServer(); err != nil {
		return fmt.Errorf("error running gRPC server: %w", err)
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
	} else if m.ctlPlaneProxy != nil {
		// Wait until control-plane proxy is healthy
		status, err := m.ctlPlaneProxy.WaitHealthy(runCtx)
		if err != nil || status.LocalPort == nil {
			m.log.Error("couldn't establish control-plane proxy", "error", err)
			cancel()
		} else {
			// Create the tunnel API client
			proxyAddress := fmt.Sprintf("localhost:%d", *status.LocalPort)
			m.log.Info("control-plane proxy is connected", "local-addr", proxyAddress)

			if err := m.setTunnelAPIClient(proxyAddress); err != nil {
				m.log.Error("couldn't create tunnel-api client", "error", err)
				cancel()
			}
		}
	}

	// Start devbox session manager
	m.devboxSessionMgr.Start(runCtx)

	// Run the sandboxes watcher
	sbmWatcher.run(runCtx, m.tunAPIClient)

	// Wait until termination
	<-runCtx.Done()

	// Check if shutdown was triggered by devbox session release
	if m.devboxSessionMgr.WasSessionReleased() {
		m.log.Info("Devbox session was released, entering deadend state")
		m.sessionReleased = true

		// Shutdown root manager (tunnel, localnet, etchosts)
		m.shutdownRootManager()

		// Stop all active work but keep gRPC server running
		sbmWatcher.stop()
		m.devboxSessionMgr.Stop(ctx)
		if m.portForward != nil {
			m.portForward.Close()
		} else if m.ctlPlaneProxy != nil {
			m.ctlPlaneProxy.Close(ctx)
		}

		// Keep gRPC server running in deadend state
		// Wait forever (until process is killed externally)
		select {}
	}

	// Normal shutdown
	m.log.Info("Shutting down")
	m.grpcServer.GracefulStop()
	sbmWatcher.stop()
	m.devboxSessionMgr.Stop(ctx)
	if m.portForward != nil {
		m.portForward.Close()
	} else if m.ctlPlaneProxy != nil {
		m.ctlPlaneProxy.Close(ctx)
	}
	return nil
}

func (m *sandboxManager) runAPIServer() error {
	addr := fmt.Sprintf(":%d", m.ciConfig.APIPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("error listening on %s: %w", addr, err)
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

// shutdownRootManager calls the root manager's Shutdown API to shut down tunnel and services
func (m *sandboxManager) shutdownRootManager() {
	if m.sbmServer == nil {
		return
	}
	rootClient := m.sbmServer.getRootClient()
	if rootClient == nil {
		m.log.Warn("Could not get root manager client to shutdown")
		return
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := rootClient.Shutdown(shutdownCtx, &rootapi.ShutdownRequest{})
	if err != nil {
		m.log.Warn("Failed to shutdown root manager", "error", err)
	} else {
		m.log.Info("Root manager shutdown requested")
	}
}
