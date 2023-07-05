package sandboxmanager

import (
	"context"
	"fmt"
	"sync"
	"time"

	cliconfig "github.com/signadot/cli/internal/config"
	commonapi "github.com/signadot/cli/internal/locald/api"
	rootapi "github.com/signadot/cli/internal/locald/api/rootmanager"
	sbapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
	clapi "github.com/signadot/libconnect/common/apiclient"
	"github.com/signadot/libconnect/common/portforward"
	"github.com/signadot/libconnect/revtun"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type grpcServer struct {
	sbapi.UnimplementedSandboxManagerAPIServer

	log *slog.Logger

	// api config for apply sandbox
	apiConfig *cliconfig.API

	// if nil, no portforward necessary
	portForward *portforward.PortForward

	// sandboxes
	sbMu             sync.Mutex
	clAPIClient      clapi.Client
	sbMonitors       map[string]*sbMonitor
	revtunClientFunc func() revtun.Client

	// rootmanager statuses
	rootMu     sync.Mutex
	rootClient rootapi.RootManagerAPIClient

	// shutdown
	shutdownCh chan struct{}
}

func newSandboxManagerGRPCServer(pf *portforward.PortForward, apiConfig *cliconfig.API,
	clAPIClient clapi.Client, rtClientFunc func() revtun.Client,
	log *slog.Logger, shutdownCh chan struct{}) *grpcServer {
	srv := &grpcServer{
		log:              log,
		apiConfig:        apiConfig,
		portForward:      pf,
		sbMonitors:       make(map[string]*sbMonitor),
		clAPIClient:      clAPIClient,
		revtunClientFunc: rtClientFunc,
		shutdownCh:       shutdownCh,
	}
	return srv
}

func (s *grpcServer) ApplySandbox(ctx context.Context, req *sbapi.ApplySandboxRequest) (*sbapi.ApplySandboxResponse, error) {
	sbSpec, err := sbapi.ToModelsSandboxSpec(req.SandboxSpec)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unable create go-sdk sandbox spec: %s", err.Error())
	}
	sb := &models.Sandbox{
		Spec: sbSpec,
	}
	s.log.Debug("api", "config", s.apiConfig)
	params := sandboxes.NewApplySandboxParams().
		WithOrgName(s.apiConfig.Org).WithSandboxName(req.Name).WithData(sb)
	result, err := s.apiConfig.Client.Sandboxes.ApplySandbox(params, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error applying sandbox to signadot api: %s", err.Error())
	}
	code := result.Code()
	switch {
	default:
		return nil, status.Errorf(codes.Unknown, "api server error: %s", result.Error())
	case code/100 == 4:
		if code == 404 {
			return nil, status.Errorf(codes.NotFound, "sandbox %q not found", req.Name)
		}
		return nil, status.Errorf(codes.InvalidArgument, "invalid sandbox %q", req.Name)
	case code/100 == 5:
		if code == 502 {
			return nil, status.Errorf(codes.Unavailable, "invalid gateway: %s", result.Error())
		}
		return nil, status.Errorf(codes.Internal, "api server error: %s", result.Error())
	case code/100 == 2:
		// success, continue below
	}

	// the api call was a success, register sandbox
	s.registerSandbox(result.Payload)

	// construct response
	grpcSandbox, err := sbapi.ToGRPCSandbox(result.Payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unable to create grpc sandbox: %s", err.Error())
	}
	resp := &sbapi.ApplySandboxResponse{}
	resp.Sandbox = grpcSandbox
	return resp, nil
}

func (s *grpcServer) registerSandbox(sb *models.Sandbox) {
	s.sbMu.Lock()
	defer s.sbMu.Unlock()
	sbm, present := s.sbMonitors[sb.Name]
	if present {
		// resolve locals
		sbm.updateLocalsSpec(sb.Spec.Local)
		return
	}
	// start watching the sandbox in the cluster
	sbm = newSBMonitor(sb.RoutingKey, s.clAPIClient, s.revtunClientFunc(), func() {
		s.sbMu.Lock()
		defer s.sbMu.Unlock()
		delete(s.sbMonitors, sb.Name)
	}, s.log.With("sandbox-routing-key", sb.RoutingKey))
	sbm.updateLocalsSpec(sb.Spec.Local)
	s.sbMonitors[sb.Name] = sbm
}

func (s *grpcServer) Status(ctx context.Context, req *sbapi.StatusRequest) (*sbapi.StatusResponse, error) {
	resp := &sbapi.StatusResponse{}
	resp.Portfoward = s.portforwardStatus()
	resp.Sandboxes = s.sbStatuses()
	resp.Hosts, resp.Localnet = s.rootStatus()
	return resp, nil
}

func (s *grpcServer) rootStatus() (*commonapi.HostsStatus, *commonapi.LocalNetStatus) {
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

func (s *grpcServer) getRootClient() rootapi.RootManagerAPIClient {
	s.rootMu.Lock()
	defer s.rootMu.Unlock()
	if s.rootClient != nil {
		return s.rootClient
	}
	grpcConn, err := grpc.Dial(
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

func (s *grpcServer) portforwardStatus() *commonapi.PortForwardStatus {
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

func (s *grpcServer) sbStatuses() []*commonapi.SandboxStatus {
	s.sbMu.Lock()
	defer s.sbMu.Unlock()
	res := make([]*commonapi.SandboxStatus, 0, len(s.sbMonitors))
	for name, sbM := range s.sbMonitors {
		sbmStatus := sbM.getStatus()
		grpcStatus := &commonapi.SandboxStatus{
			Name:           name,
			RoutingKey:     sbM.routingKey,
			LocalWorkloads: make([]*commonapi.SandboxStatus_LocalWorkload, len(sbmStatus.ExternalWorkloads)),
		}
		for i, xw := range sbmStatus.ExternalWorkloads {
			lwStatus := &commonapi.SandboxStatus_LocalWorkload{
				Name: xw.Name,
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

func (s *grpcServer) Shutdown(ctx context.Context, req *sbapi.ShutdownRequest) (*sbapi.ShutdownResponse, error) {
	select {
	case <-s.shutdownCh:
	default:
		close(s.shutdownCh)
	}
	return &sbapi.ShutdownResponse{}, nil
}
