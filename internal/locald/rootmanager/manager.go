package rootmanager

import (
	"fmt"
	"net"

	"github.com/signadot/cli/internal/config"
	rootapi "github.com/signadot/cli/internal/locald/api/rootmanager"
	connectcfg "github.com/signadot/libconnect/config"
	"google.golang.org/grpc"
)

type rootManager struct {
	connConfig *connectcfg.ConnectionConfig
	listenPort uint16
	grpcServer *grpc.Server
}

func NewRootManager(cfg *config.LocalDaemon, args []string) (*rootManager, error) {
	if err := cfg.InitLocalDaemon(); err != nil {
		return nil, err
	}

	grpcServer := grpc.NewServer()
	rootapi.RegisterRootManagerAPIServer(grpcServer, &rootServer{})

	return &rootManager{
		grpcServer: grpcServer,
		connConfig: cfg.ConnectInvocationConfig.ConnectionConfig,
		listenPort: cfg.ConnectInvocationConfig.LocalNetPort,
	}, nil
}

func (m *rootManager) Run() error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", m.listenPort))
	if err != nil {
		return err
	}
	return m.grpcServer.Serve(ln)
}
