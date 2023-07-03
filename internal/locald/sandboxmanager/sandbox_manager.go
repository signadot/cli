package sandboxmanager

import (
	"context"
	"fmt"
	"net"
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

	grpcServer := grpc.NewServer()
	sbMgr := &sandboxManager{
		log:          log.With("locald-component", "sandbox-manager"),
		apiPort:      cfg.ConnectInvocationConfig.APIPort,
		proxyAddress: proxyAddress,
		grpcServer:   grpcServer,
		connConfig:   connConfig,
		portForward:  portForward,
		localdConfig: cfg,
	}

	sbmSrv := newSandboxManagerGRPCServer(portForward, connConfig.ProxyAddress,
		ciConfig.API, clapiClient, sbMgr.revtunClient, log)
	sandboxmanager.RegisterSandboxManagerAPIServer(grpcServer, sbmSrv)

	return &sandboxManager{
		log:          log,
		apiPort:      cfg.ConnectInvocationConfig.APIPort,
		grpcServer:   grpcServer,
		connConfig:   connConfig,
		localdConfig: cfg,
	}, nil
}

func (m *sandboxManager) Run() error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", m.apiPort))
	if err != nil {
		return err
	}
	return m.grpcServer.Serve(ln)
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
