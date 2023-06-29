package sandboxmanager

import (
	"context"
	"fmt"

	sbmgrapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	"github.com/signadot/go-sdk/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Apply is for client parts of the cli (everything else in locald is
// server/daemon side)
func Apply(ctx context.Context, org, name string, spec *models.SandboxSpec) (*models.Sandbox, error) {
	grpcSpec, err := sbmgrapi.ToGRPCSandboxSpec(spec)
	if err != nil {
		return nil, err
	}
	req := &sbmgrapi.ApplySandboxRequest{
		Name:        name,
		SandboxSpec: grpcSpec,
	}
	conn, err := grpc.Dial("localhost:6666",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("unable to dial sandboxmanager: %w", err)
	}
	cli := sbmgrapi.NewSandboxManagerAPIClient(conn)
	grpcResp, err := cli.ApplySandbox(ctx, req)
	if err != nil {
		return nil, err
	}
	return sbmgrapi.ToModelsSandbox(grpcResp.Sandbox)
}
