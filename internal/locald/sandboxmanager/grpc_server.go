package sandboxmanager

import (
	"context"
	"fmt"
	"sync"

	cliconfig "github.com/signadot/cli/internal/config"
	commonapi "github.com/signadot/cli/internal/locald/api"
	sbmgrpc "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
	clapi "github.com/signadot/libconnect/common/apiclient"
	"github.com/signadot/libconnect/common/portforward"
	"github.com/signadot/libconnect/revtun"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcServer struct {
	sbmgrpc.UnimplementedSandboxManagerAPIServer

	log *slog.Logger

	// api config for apply sandbox
	apiConfig *cliconfig.API

	// if nil, no portforward necessary
	portForward *portforward.PortForward
	// if pf nil, use this instead
	clusterProxyAddr string

	// sandboxes
	sbMu             sync.Mutex
	clAPIClient      clapi.Client
	sbMonitors       map[string]*sbMonitor
	revtunClientFunc func() revtun.Client

	// rootmanager statuses
}

func newSandboxManagerGRPCServer(pf *portforward.PortForward, clProxyAddr string, apiConfig *cliconfig.API, clAPIClient clapi.Client, rtClientFunc func() revtun.Client, log *slog.Logger) *grpcServer {
	srv := &grpcServer{
		log:              log,
		apiConfig:        apiConfig,
		portForward:      pf,
		clusterProxyAddr: clProxyAddr,
		sbMonitors:       make(map[string]*sbMonitor),
		clAPIClient:      clAPIClient,
		revtunClientFunc: rtClientFunc,
	}
	return srv
}

func (s *grpcServer) ApplySandbox(ctx context.Context, req *sbmgrpc.ApplySandboxRequest) (*sbmgrpc.ApplySandboxResponse, error) {
	sbSpec, err := sbmgrpc.ToModelsSandboxSpec(req.SandboxSpec)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unable create go-sdk sandbox spec: %s", err.Error())
	}
	sb := &models.Sandbox{
		Spec: sbSpec,
	}
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
	grpcSandbox, err := sbmgrpc.ToGRPCSandbox(result.Payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unable to create grpc sandbox: %s", err.Error())
	}
	resp := &sbmgrpc.ApplySandboxResponse{}
	resp.Sandbox = grpcSandbox
	return resp, nil
}

func (s *grpcServer) registerSandbox(sb *models.Sandbox) {
	s.sbMu.Lock()
	defer s.sbMu.Unlock()
	sbm, present := s.sbMonitors[sb.Name]
	if present {
		// resolve locals
		sbm.reconcileLocals(sb.Spec.Local)
		return
	}
	sbm = newSBMonitor(sb.RoutingKey, s.clAPIClient, s.revtunClientFunc, func() {
		s.sbMu.Lock()
		defer s.sbMu.Unlock()
		delete(s.sbMonitors, sb.Name)
	}, s.log)
	sbm.reconcileLocals(sb.Spec.Local)
	s.sbMonitors[sb.Name] = sbm
}

func (s *grpcServer) Status(ctx context.Context, req *sbmgrpc.StatusRequest) (*sbmgrpc.StatusResponse, error) {
	resp := &sbmgrpc.StatusResponse{}
	resp.Portfoward = s.portforwardStatus()
	resp.Sandboxes = s.sbStatuses()
	return resp, nil
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
