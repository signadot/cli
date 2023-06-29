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
	localdConfig *config.LocalDaemon
	connConfig   *connectcfg.ConnectionConfig
	grpcServer   *grpc.Server
}

func NewSandboxManager(cfg *config.LocalDaemon, args []string, log *slog.Logger) (*sandboxManager, error) {
	connConfig := cfg.ConnectInvocationConfig.ConnectionConfig
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
	clapiClient, err := getClusterAPIClient(portForward, connConfig.ProxyAddress)

	if err != nil {
		return nil, err
	}
	log.Debug("got cluster api client")

	grpcServer := grpc.NewServer()
	sbMgr := &sandboxManager{
		log:          log,
		apiPort:      cfg.ConnectInvocationConfig.APIPort,
		grpcServer:   grpcServer,
		connConfig:   connConfig,
		localdConfig: cfg,
	}

	sbmSrv := newSandboxManagerGRPCServer(portForward, cfg.ConnectInvocationConfig.ConnectionConfig.ProxyAddress,
		cfg.API, clapiClient, sbMgr.revtunClient, log)
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
		User: connConfig.KubeContext,
		ErrFunc: func(e error) {
			m.log.Error("revtun.errfunc", "error", e)
		},
	}
	switch connConfig.Inbound.Protocol {
	case connectcfg.XAPInboundProtocol:
		return xaprevtun.NewClient(rtClientConfig, "")
	case connectcfg.SSHInboundProtocol:
		return sshrevtun.NewClient(rtClientConfig, nil)
	default:
		// already validated
		panic(fmt.Errorf("invalid inbound protocol: %s", connConfig.Inbound.Protocol))
	}
	return nil
}

func getClusterAPIClient(pf *portforward.PortForward, proxyAddress string) (clapiclient.Client, error) {
	if pf == nil {
		return clapiclient.NewClient(proxyAddress)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	st, err := pf.WaitHealthy(ctx)
	if err != nil || st.LocalPort == nil {
		return nil, fmt.Errorf("error getting tunnel api client: %w: portforward not ready in time", err)
	}
	return clapiclient.NewClient(fmt.Sprintf(":%d", *st.LocalPort))
}
