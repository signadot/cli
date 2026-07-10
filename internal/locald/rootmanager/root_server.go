package rootmanager

import (
	"context"
	"net"
	"sort"
	"sync"

	commonapi "github.com/signadot/cli/internal/locald/api"
	rootapi "github.com/signadot/cli/internal/locald/api/rootmanager"
	"github.com/signadot/libconnect/common/apiclient"
	"github.com/signadot/libconnect/fwdtun/etchosts"
	"github.com/signadot/libconnect/fwdtun/ipmap"
	"github.com/signadot/libconnect/fwdtun/localdns"
	"github.com/signadot/libconnect/fwdtun/localnet"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type rootServer struct {
	rootapi.UnimplementedRootManagerAPIServer

	mu          sync.RWMutex
	localnetSVC *localnet.Service
	etcHostsSVC *etchosts.EtcHosts
	// localDNSSVC is the alternative to etcHostsSVC, selected by --local-dns.
	// The CLI owns the injected tunnel-api client and closes it on teardown.
	localDNSSVC       *localdns.Service
	localDNSAPIClient apiclient.Client
	// ipMap is the shared host->address allocator that backs whichever
	// name-resolution service is running; it is the source of truth for
	// GetHosts in both /etc/hosts and local-DNS modes.
	ipMap      *ipmap.IPMap
	shutdownCh chan struct{}
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

	// Local DNS
	var localDNSSt *commonapi.LocalDNSStatus
	if s.localDNSSVC != nil {
		status := s.localDNSSVC.Status()

		var lastErrortime *timestamppb.Timestamp
		if status.LastErrorTime != nil {
			lastErrortime = timestamppb.New(*status.LastErrorTime)
		}
		var lastRefresh *timestamppb.Timestamp
		if status.LastRefresh != nil {
			lastRefresh = timestamppb.New(*status.LastRefresh)
		}
		localDNSSt = &commonapi.LocalDNSStatus{
			Health: &commonapi.ServiceHealth{
				Healthy:         status.Healthy,
				ErrorCount:      uint32(status.ErrorCount),
				LastErrorReason: status.LastErrorReason,
				LastErrorTime:   lastErrortime,
			},
			Mode:              status.Mode,
			BindAddr:          status.BindAddr,
			Suffixes:          status.Suffixes,
			Upstreams:         status.Upstreams,
			RecordCount:       uint32(status.RecordCount),
			LastRefresh:       lastRefresh,
			ResolvConfManaged: status.ResolvConfManaged,
			Warning:           status.Warning,
		}
	}

	resp := &rootapi.StatusResponse{
		Localnet: localnetSt,
		Hosts:    etcHostsSt,
		LocalDns: localDNSSt,
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

func (s *rootServer) GetHosts(ctx context.Context, req *rootapi.GetHostsRequest) (*rootapi.GetHostsResponse, error) {
	s.mu.RLock()
	ipMap := s.ipMap
	s.mu.RUnlock()

	resp := &rootapi.GetHostsResponse{}
	if ipMap == nil {
		// No name-resolution service has started yet (e.g. tunnel-proxy link
		// still coming up): report an empty set rather than an error.
		return resp, nil
	}

	entries := ipMap.Entries()
	resp.Entries = make([]*commonapi.HostEntry, 0, len(entries))
	for name, ips := range entries {
		ip := pickIP(ips)
		if ip == "" {
			// A host with no assigned address is not resolvable; skip it.
			continue
		}
		resp.Entries = append(resp.Entries, &commonapi.HostEntry{
			Name: name,
			Ip:   ip,
		})
	}
	// Stable, name-sorted output so callers (and `signadot local hosts`) get a
	// deterministic listing.
	sort.Slice(resp.Entries, func(i, j int) bool {
		return resp.Entries[i].Name < resp.Entries[j].Name
	})
	return resp, nil
}

// pickIP returns the single address to report for a host: the IPv4 address when
// one is present, otherwise the first IPv6 address (covering an IPv6-only
// allocation). It returns "" when the host has no address. A host is assigned
// one virtual address in practice; the preference only makes the choice
// deterministic should it ever carry both families.
func pickIP(ips []net.IP) string {
	var fallback string
	for _, ip := range ips {
		if ip.To4() != nil {
			return ip.String()
		}
		if fallback == "" {
			fallback = ip.String()
		}
	}
	return fallback
}

func (s *rootServer) setIPMap(ipMap *ipmap.IPMap) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ipMap = ipMap
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

func (s *rootServer) setLocalDNSService(svc *localdns.Service, apiClient apiclient.Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.localDNSSVC = svc
	s.localDNSAPIClient = apiClient
}

func (s *rootServer) getLocalDNSService() (*localdns.Service, apiclient.Client) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.localDNSSVC, s.localDNSAPIClient
}
