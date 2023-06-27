package locald

import (
	"fmt"
	"net"

	"google.golang.org/grpc"

	"github.com/signadot/cli/internal/config"
	connectcfg "github.com/signadot/libconnect/config"
)

type sandboxManager struct {
	localdConfig *config.LocalDaemon
	connConfig   *connectcfg.ConnectionConfig
	grpcServer   *grpc.Server
}

func newSandboxManager(cfg *config.LocalDaemon, args []string) (*sandboxManager, error) {
	grpcServer := grpc.NewServer()
	connConfig, err := cfg.GetConnectionConfig(cfg.Cluster)
	if err != nil {
		return nil, err
	}
	_ = connConfig
	// TODO setup everything

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
