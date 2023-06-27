package locald

import (
	"context"
	"fmt"
	"net"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/locald/api/sandboxmanager"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"

	"github.com/signadot/libconnect/common/portforward"
	connectcfg "github.com/signadot/libconnect/config"
)

type sandboxManager struct {
	localdConfig *config.LocalDaemon
	connConfig   *connectcfg.ConnectionConfig
	grpcServer   *grpc.Server
}

func newSandboxManager(cfg *config.LocalDaemon, args []string) (*sandboxManager, error) {
	connConfig, err := cfg.GetConnectionConfig(cfg.Cluster)
	if err != nil {
		return nil, err
	}
	restConfig, err := connConfig.GetRESTConfig()
	if err != nil {
		return nil, err
	}
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	var portForward *portforward.PortForward
	if connConfig.Type == connectcfg.PortForwardLinkType {
		portForward = portforward.NewPortForward(context.Background(),
			restConfig, clientSet, slog.Default(), 0, "signadot", "tunnel-proxy", 1080)
	}
	grpcServer := grpc.NewServer()
	sandboxmanager.RegisterSandboxManagerAPIServer(grpcServer, &sbmServer{
		portForward: portForward,
	})

	return &sandboxManager{
		grpcServer:   grpcServer,
		connConfig:   connConfig,
		localdConfig: cfg,
	}, nil
}

func (m *sandboxManager) Run() error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", m.localdConfig.Port))
	if err != nil {
		return err
	}
	return m.grpcServer.Serve(ln)
}
