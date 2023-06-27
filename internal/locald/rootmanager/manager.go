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
	localdConfig *config.LocalDaemon
	connConfig   *connectcfg.ConnectionConfig
	grpcServer   *grpc.Server
}

func NewRootManager(cfg *config.LocalDaemon, args []string) (*rootManager, error) {
	connConfig, err := cfg.GetConnectionConfig(cfg.Cluster)
	if err != nil {
		return nil, err
	}

	grpcServer := grpc.NewServer()
	rootapi.RegisterRootManagerAPIServer(grpcServer, &rootServer{})

	return &rootManager{
		grpcServer:   grpcServer,
		connConfig:   connConfig,
		localdConfig: cfg,
	}, nil
}

func (m *rootManager) Run() error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", m.localdConfig.Port))
	if err != nil {
		return err
	}
	return m.grpcServer.Serve(ln)
}
