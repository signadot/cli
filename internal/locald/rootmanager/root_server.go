package rootmanager

import (
	"context"

	rootapi "github.com/signadot/cli/internal/locald/api/rootmanager"
)

type rootServer struct {
	rootapi.UnimplementedRootManagerAPIServer
}

var _ rootapi.RootManagerAPIServer = &rootServer{}

func (s *rootServer) Status(ctx context.Context, req *rootapi.StatusRequest) (*rootapi.StatusResponse, error) {
	resp := &rootapi.StatusResponse{}
	return resp, nil
}
