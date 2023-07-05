package sandboxmanager

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/locald/api/sandboxmanager"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"

	clapiclient "github.com/signadot/libconnect/common/apiclient"
	"github.com/signadot/libconnect/common/portforward"
	connectcfg "github.com/signadot/libconnect/config"
	"github.com/signadot/libconnect/revtun"
	"github.com/signadot/libconnect/revtun/sshrevtun"
	"github.com/signadot/libconnect/revtun/xaprevtun"
)

type sandboxManager struct {
	log          *slog.Logger
	apiPort      uint16
	proxyAddress string
	localdConfig *config.LocalDaemon
	connConfig   *connectcfg.ConnectionConfig
	grpcServer   *grpc.Server
	portForward  *portforward.PortForward
	shutdownCh   chan struct{}
}

func NewSandboxManager(cfg *config.LocalDaemon, args []string, log *slog.Logger) (*sandboxManager, error) {
	ciConfig := cfg.ConnectInvocationConfig
	connConfig := ciConfig.ConnectionConfig
	restConfig, err := connConfig.GetRESTConfig()
	if err != nil {
		return nil, err
	}
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	var (
		portForward *portforward.PortForward
	)
	switch connConfig.Type {
	case connectcfg.PortForwardLinkType:
		portForward = portforward.NewPortForward(context.Background(),
			restConfig, clientSet, slog.Default(), 0, "signadot", "tunnel-proxy", 1080)
	}
	proxyAddress, err := getProxyAddress(portForward, connConfig.ProxyAddress)
	if err != nil {
		return nil, err
	}
	clapiClient, err := clapiclient.NewClient(proxyAddress, clapiclient.DefaultClientKeepaliveParams())
	if err != nil {
		return nil, err
	}
	log.Debug("got cluster api client")

	shutdownCh := make(chan struct{})
	grpcServer := grpc.NewServer()
	sbMgr := &sandboxManager{
		log:          log,
		apiPort:      cfg.ConnectInvocationConfig.APIPort,
		proxyAddress: proxyAddress,
		grpcServer:   grpcServer,
		connConfig:   connConfig,
		portForward:  portForward,
		localdConfig: cfg,
		shutdownCh:   shutdownCh,
	}

	sbmSrv := newSandboxManagerGRPCServer(portForward, ciConfig.API, clapiClient,
		sbMgr.revtunClient, log, shutdownCh)
	sandboxmanager.RegisterSandboxManagerAPIServer(grpcServer, sbmSrv)
	return sbMgr, nil
}

func (m *sandboxManager) Run() error {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", m.apiPort))
	if err != nil {
		return err
	}
	go m.grpcServer.Serve(ln)

	// Wait until termination
	select {
	case <-sigs:
	case <-m.shutdownCh:
	}

	// Clean up
	m.log.Info("Shutting down")
	m.grpcServer.GracefulStop()
	return nil
}

func (m *sandboxManager) revtunClient() revtun.Client {
	connConfig := m.localdConfig.ConnectInvocationConfig.ConnectionConfig
	rtClientConfig := &revtun.ClientConfig{
		User:       connConfig.KubeContext,
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

func getProxyAddress(pf *portforward.PortForward, proxyAddress string) (string, error) {
	if pf == nil {
		return proxyAddress, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	st, err := pf.WaitHealthy(ctx)
	if err != nil || st.LocalPort == nil {
		return "", fmt.Errorf("error getting tunnel api client: %w: portforward not ready in time", err)
	}
	return fmt.Sprintf("localhost:%d", *st.LocalPort), nil
}
