package sandboxmanager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"log/slog"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/devbox"
	commonapi "github.com/signadot/cli/internal/locald/api"
	rootapi "github.com/signadot/cli/internal/locald/api/rootmanager"
	sbapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	"github.com/signadot/libconnect/apiv1"
	"github.com/signadot/libconnect/common/controlplaneproxy"
	"github.com/signadot/libconnect/common/portforward"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type sbmServer struct {
	sbapi.UnimplementedSandboxManagerAPIServer

	log *slog.Logger
	oiu *operatorInfoUpdater

	// runtime config
	ciConfig *config.ConnectInvocationConfig

	// if nil, no portforward necessary
	portForward *portforward.PortForward

	// if nil, no control-plane proxy necessary
	ctlPlaneProxy *controlplaneproxy.Proxy

	// sandboxes
	sbmWatcher *sbmWatcher

	// rootmanager statuses
	rootMu     sync.Mutex
	rootClient rootapi.RootManagerAPIClient

	// shutdown
	shutdownCh chan struct{}

	// devbox session manager
	devboxSessionMgr *devbox.SessionManager
}

func newSandboxManagerGRPCServer(log *slog.Logger, ciConfig *config.ConnectInvocationConfig,
	portForward *portforward.PortForward, ctlPlaneProxy *controlplaneproxy.Proxy,
	sbmWatcher *sbmWatcher, oiu *operatorInfoUpdater,
	shutdownCh chan struct{}, devboxSessionMgr *devbox.SessionManager) *sbmServer {
	srv := &sbmServer{
		log:              log,
		oiu:              oiu,
		ciConfig:         ciConfig,
		portForward:      portForward,
		ctlPlaneProxy:    ctlPlaneProxy,
		sbmWatcher:       sbmWatcher,
		shutdownCh:       shutdownCh,
		devboxSessionMgr: devboxSessionMgr,
	}
	return srv
}

func (s *sbmServer) Status(ctx context.Context, req *sbapi.StatusRequest) (*sbapi.StatusResponse, error) {
	// make a local copy
	sbConfig := *s.ciConfig

	grpcCIConfig, err := sbapi.ToGRPCCIConfig(&sbConfig)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unable to create grpc ci-config: %s", err.Error())
	}

	resp := &sbapi.StatusResponse{
		CiConfig:          grpcCIConfig,
		OperatorInfo:      s.oiu.Get(),
		Portforward:       s.portForwardStatus(),
		ControlPlaneProxy: s.controlPlaneProxyStatus(),
		Watcher:           s.watcherStatus(),
		Sandboxes:         s.sbStatuses(),
		DevboxSession:     s.devboxSessionStatus(),
	}
	resp.Hosts, resp.Localnet = s.rootStatus()
	return resp, nil
}

func (s *sbmServer) Shutdown(ctx context.Context, req *sbapi.ShutdownRequest) (*sbapi.ShutdownResponse, error) {
	select {
	case <-s.shutdownCh:
	default:
		close(s.shutdownCh)
	}
	return &sbapi.ShutdownResponse{}, nil
}

func (s *sbmServer) GetResourceOutputs(ctx context.Context, req *sbapi.GetResourceOutputsRequest) (*sbapi.GetResourceOutputsResponse, error) {
	tac := s.sbmWatcher.tunAPIClient
	tunReq := &apiv1.GetResourceOutputsRequest{
		SandboxRoutingKey: req.SandboxRoutingKey,
	}
	resp, err := tac.GetResourceOutputs(ctx, tunReq)
	if err != nil {
		// TODO: if method does not exist
		return nil, fmt.Errorf("tunnel-api error getting resource outputs: %w", err)
	}
	res := &sbapi.GetResourceOutputsResponse{}
	for _, rv := range resp.ResourceValues {
		resRVs := &sbapi.ResourceOutputs{}
		resRVs.ResourceName = rv.ResourceName
		for _, out := range rv.Outputs {
			resRVs.Outputs = append(resRVs.Outputs, &sbapi.ResourceOutputItem{
				Key:   out.OutputKey,
				Value: out.OutputValue,
			})
		}
		res.ResourceOutputs = append(res.ResourceOutputs, resRVs)
	}
	return res, nil
}

func (s *sbmServer) rootStatus() (*commonapi.HostsStatus, *commonapi.LocalNetStatus) {
	if !s.ciConfig.WithRootManager {
		// We are running without a root manager
		return nil, nil
	}

	rootClient := s.getRootClient()
	if rootClient == nil {
		s.log.Debug("no root client available for rootStatus()")
		return &commonapi.HostsStatus{}, &commonapi.LocalNetStatus{}
	}
	req := &rootapi.StatusRequest{}
	ctx, cancel := context.WithTimeout(context.Background(),
		3*time.Second)
	defer cancel()
	resp, err := rootClient.Status(ctx, req)
	if err != nil {
		s.log.Error("error getting status from root manager", "error", err)
		return &commonapi.HostsStatus{}, &commonapi.LocalNetStatus{}
	}
	return resp.Hosts, resp.Localnet
}

func (s *sbmServer) getRootClient() rootapi.RootManagerAPIClient {
	s.rootMu.Lock()
	defer s.rootMu.Unlock()
	if s.rootClient != nil {
		return s.rootClient
	}
	grpcConn, err := grpc.NewClient(
		"127.0.0.1:6667",
		grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		s.log.Debug("couldn't get root client", "error", err)
		return nil
	}
	s.log.Debug("got root client")
	s.rootClient = rootapi.NewRootManagerAPIClient(grpcConn)
	return s.rootClient
}

func (s *sbmServer) portForwardStatus() *commonapi.PortForwardStatus {
	grpcPFStatus := &commonapi.PortForwardStatus{}
	if s.portForward == nil {
		return grpcPFStatus
	}
	pfst := s.portForward.Status()
	grpcPFStatus.Health = commonapi.ToGRPCServiceHealth(&pfst.ServiceHealth)
	if pfst.LocalPort != nil && pfst.Healthy {
		grpcPFStatus.LocalAddress = fmt.Sprintf(":%d", *pfst.LocalPort)
	}
	return grpcPFStatus
}

func (s *sbmServer) controlPlaneProxyStatus() *commonapi.ControlPlaneProxyStatus {
	grpcCPPStatus := &commonapi.ControlPlaneProxyStatus{}
	if s.ctlPlaneProxy == nil {
		return grpcCPPStatus
	}
	st := s.ctlPlaneProxy.Status()
	grpcCPPStatus.Health = commonapi.ToGRPCServiceHealth(&st.ServiceHealth)
	if st.LocalPort != nil && st.Healthy {
		grpcCPPStatus.LocalAddress = fmt.Sprintf(":%d", *st.LocalPort)
	}
	return grpcCPPStatus
}

func (s *sbmServer) watcherStatus() *commonapi.WatcherStatus {
	return &commonapi.WatcherStatus{
		Health: commonapi.ToGRPCServiceHealth(s.sbmWatcher.getStatus()),
	}
}

func (s *sbmServer) devboxSessionStatus() *commonapi.DevboxSessionStatus {
	if s.devboxSessionMgr == nil {
		return &commonapi.DevboxSessionStatus{
			Healthy: false,
		}
	}

	healthy, sessionReleased, devboxID, sessionID, lastErrorTime, lastError := s.devboxSessionMgr.GetStatus()

	status := &commonapi.DevboxSessionStatus{
		Healthy:         healthy && !sessionReleased,
		SessionReleased: sessionReleased,
		DevboxId:        devboxID,
		SessionId:       sessionID,
	}

	if lastError != nil {
		status.LastErrorReason = lastError.Error()
		if !lastErrorTime.IsZero() {
			status.LastErrorTime = timestamppb.New(lastErrorTime)
		}
	}

	return status
}

func (s *sbmServer) sbStatuses() []*commonapi.SandboxStatus {
	sandboxes := s.sbmWatcher.getSandboxes()

	res := make([]*commonapi.SandboxStatus, 0, len(sandboxes))
	for _, sbmStatus := range sandboxes {
		grpcStatus := &commonapi.SandboxStatus{
			Name:           sbmStatus.SandboxName,
			RoutingKey:     sbmStatus.RoutingKey,
			LocalWorkloads: make([]*commonapi.SandboxStatus_LocalWorkload, len(sbmStatus.ExternalWorkloads)),
		}
		for i, xw := range sbmStatus.ExternalWorkloads {
			portMapping := []*commonapi.SandboxStatus_BaselineToLocal{}
			for _, pm := range xw.WorkloadPortMapping {
				portMapping = append(portMapping, &commonapi.SandboxStatus_BaselineToLocal{
					BaselinePort: pm.BaselinePort,
					LocalAddress: pm.LocalAddress,
				})
			}

			lwStatus := &commonapi.SandboxStatus_LocalWorkload{
				Name: xw.Name,
				Baseline: &commonapi.SandboxStatus_Baseline{
					ApiVersion: xw.Baseline.ApiVersion,
					Kind:       xw.Baseline.Kind,
					Namespace:  xw.Baseline.Namespace,
					Name:       xw.Baseline.Name,
				},
				WorkloadPortMapping: portMapping,
				TunnelHealth: &commonapi.ServiceHealth{
					Healthy: xw.Connected,
					// TODO plumb health into commonapi
				},
			}
			grpcStatus.LocalWorkloads[i] = lwStatus
		}
		res = append(res, grpcStatus)
	}
	return res
}
