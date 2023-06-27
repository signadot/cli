package sandboxmanager

import (
	"context"
	"fmt"

	commonapi "github.com/signadot/cli/internal/locald/api"
	sbmgrpc "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	"github.com/signadot/libconnect/common/portforward"
)

type sbmServer struct {
	sbmgrpc.UnimplementedSandboxManagerAPIServer

	// if nil, no portforward necessary
	portForward *portforward.PortForward
}

func (s *sbmServer) Status(ctx context.Context, req *sbmgrpc.StatusRequest) (*sbmgrpc.StatusResponse, error) {
	resp := &sbmgrpc.StatusResponse{}
	resp.Portfoward = &commonapi.PortForwardStatus{}
	if s.portForward != nil {
		st := s.portForward.Status()
		resp.Portfoward.Health = commonapi.ToGRPCServiceHealth(&st.ServiceHealth)
		if st.LocalPort != nil && st.Healthy {
			resp.Portfoward.LocalAddress = fmt.Sprintf(":%d", *st.LocalPort)
		}
	}
	return resp, nil
}
