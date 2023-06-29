package rootmanager

import (
	"context"
	"sync"

	commonapi "github.com/signadot/cli/internal/locald/api"
	rootapi "github.com/signadot/cli/internal/locald/api/rootmanager"
	"github.com/signadot/libconnect/fwdtun/etchosts"
	"github.com/signadot/libconnect/fwdtun/localnet"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type rootServer struct {
	rootapi.UnimplementedRootManagerAPIServer

	mu          sync.RWMutex
	localnetSVC *localnet.Service
	etcHostsSVC *etchosts.EtcHosts
	shutdownCh  chan struct{}
}

var _ rootapi.RootManagerAPIServer = &rootServer{}

func (s *rootServer) Status(ctx context.Context, req *rootapi.StatusRequest) (*rootapi.StatusResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Localnet
	var localnetSt *commonapi.LocalNetStatus
	if s.localnetSVC != nil {
		// Get localnet status
		status := s.localnetSVC.Status()

		// Convert it to gRPC response
		var lastErrortime *timestamppb.Timestamp
		if status.LastErrorTime != nil {
			lastErrortime = timestamppb.New(*status.LastErrorTime)
		}
		localnetSt = &commonapi.LocalNetStatus{
			Health: &commonapi.ServiceHealth{
				Healthy:         status.Healthy,
				ErrorCount:      uint32(status.ErrorCount),
				LastErrorReason: status.LastErrorReason,
				LastErrorTime:   lastErrortime,
			},
			Cidrs:         status.CIDRs,
			ExcludedCidrs: status.ExcludedCIDRs,
		}
	}

	// Etc Hosts
	var etcHostsSt *commonapi.HostsStatus
	if s.etcHostsSVC != nil {
		// Get etc hosts status
		status := s.etcHostsSVC.Status()

		// Convert it to gRPC response
		var lastErrortime *timestamppb.Timestamp
		if status.LastErrorTime != nil {
			lastErrortime = timestamppb.New(*status.LastErrorTime)
		}
		var lastUpdateTime *timestamppb.Timestamp
		if status.LastUpdateTime != nil {
			lastUpdateTime = timestamppb.New(*status.LastUpdateTime)
		}
		etcHostsSt = &commonapi.HostsStatus{
			Health: &commonapi.ServiceHealth{
				Healthy:         status.Healthy,
				ErrorCount:      uint32(status.ErrorCount),
				LastErrorReason: status.LastErrorReason,
				LastErrorTime:   lastErrortime,
			},
			NumHosts:       uint32(status.Hosts),
			NumUpdates:     uint32(status.Updates),
			LastUpdateTime: lastUpdateTime,
		}
	}

	resp := &rootapi.StatusResponse{
		Localnet: localnetSt,
		Hosts:    etcHostsSt,
	}
	return resp, nil
}

func (s *rootServer) Shutdown(ctx context.Context, req *rootapi.ShutdownRequest) (*rootapi.ShutdownResponse, error) {
	select {
	case <-s.shutdownCh:
	default:
		close(s.shutdownCh)
	}
	return &rootapi.ShutdownResponse{}, nil
}

func (s *rootServer) setLocalnetService(localnetSVC *localnet.Service) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.localnetSVC = localnetSVC
}

func (s *rootServer) getLocalnetService() *localnet.Service {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.localnetSVC
}

func (s *rootServer) setEtcHostsService(etcHostsSVC *etchosts.EtcHosts) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.etcHostsSVC = etcHostsSVC
}

func (s *rootServer) getEtcHostsService() *etchosts.EtcHosts {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.etcHostsSVC
}
