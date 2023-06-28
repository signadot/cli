package sandboxmanager

import (
	"context"
	"errors"
	"fmt"
	"sync"

	cliconfig "github.com/signadot/cli/internal/config"
	commonapi "github.com/signadot/cli/internal/locald/api"
	sbmgrpc "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
	clapi "github.com/signadot/libconnect/common/apiclient"
	"github.com/signadot/libconnect/common/portforward"
	"golang.org/x/exp/slog"
)

type sbmServer struct {
	sbmgrpc.UnimplementedSandboxManagerAPIServer

	log *slog.Logger

	// api config for apply sandbox
	apiConfig *cliconfig.API

	// if nil, no portforward necessary
	portForward *portforward.PortForward
	// if pf nil, use this instead
	clusterProxyAddr string

	// sandboxes
	sbMu            sync.Mutex
	clapiClient     clapi.Client
	clapiClientErrC chan error
	sbMonitors      map[string]*sbMonitor

	// rootmanager statuses
}

func newSBMServer(pf *portforward.PortForward, clProxyAddr string, apiConfig *cliconfig.API, log *slog.Logger) *sbmServer {
	srv := &sbmServer{
		log:              log,
		apiConfig:        apiConfig,
		portForward:      pf,
		clusterProxyAddr: clProxyAddr,
		sbMonitors:       make(map[string]*sbMonitor),
		clapiClientErrC:  make(chan error),
	}
	go srv.monitorSandboxes()
	return srv
}

func (s *sbmServer) monitorSandboxes() {
	for {
		func() {
			s.sbMu.Lock()
			defer s.sbMu.Unlock()
		}()
		select {
		case err := <-s.clapiClientErrC:
			s.log.Error("tunnel api client error", "error", err)
			// create a new client
		}
	}
}

func (s *sbmServer) ApplySandbox(ctx context.Context, req *sbmgrpc.ApplySandboxRequest) (*sbmgrpc.ApplySandboxResponse, error) {
	sbSpec, err := sbmgrpc.ToModelsSandboxSpec(req.SandboxSpec)
	if err != nil {
		return nil, err
	}
	sb := &models.Sandbox{
		Spec: sbSpec,
	}
	params := sandboxes.NewApplySandboxParams().
		WithOrgName(s.apiConfig.Org).WithSandboxName(req.Name).WithData(sb)
	result, err := s.apiConfig.Client.Sandboxes.ApplySandbox(params, nil)
	if err != nil {
		return nil, err
	}
	if !result.IsSuccess() {
		if result.IsClientError() {
			// TODO grpc errors
			return nil, errors.New(result.Error())

		} else if result.IsServerError() {
			// TODO grpc errors
			return nil, errors.New(result.Error())
		} else {
			return nil, errors.New(result.Error())
		}
	}
	// the api call was a success, register sandbox
	s.registerSandbox(result.Payload)

	// construct response
	grpcSandbox, err := sbmgrpc.ToGRPCSandbox(result.Payload)
	if err != nil {
		return nil, err
	}
	resp := &sbmgrpc.ApplySandboxResponse{}
	resp.Sandbox = grpcSandbox
	return resp, nil
}

func (s *sbmServer) registerSandbox(sb *models.Sandbox) {
	cond := sync.NewCond(&s.sbMu)
	cond.L.Lock()
	for s.clapiClient == nil {
		cond.Wait()
	}
	defer cond.L.Unlock()
	_, present := s.sbMonitors[sb.Name]
	if present {
		return
	}
	sbm := newSBMonitor(sb.RoutingKey, s.clapiClientErrC)
	sbm.clapiClientC <- s.clapiClient
	s.sbMonitors[sb.Name] = sbm
}

func (s *sbmServer) Status(ctx context.Context, req *sbmgrpc.StatusRequest) (*sbmgrpc.StatusResponse, error) {
	resp := &sbmgrpc.StatusResponse{}
	resp.Portfoward = s.portforwardStatus()
	resp.Sandboxes = s.sbStatuses()
	return resp, nil
}

func (s *sbmServer) portforwardStatus() *commonapi.PortForwardStatus {
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

func (s *sbmServer) sbStatuses() []*commonapi.SandboxStatus {
	res := []*commonapi.SandboxStatus{}
	return res
}
