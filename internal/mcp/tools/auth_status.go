package tools

import (
	"bytes"
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/signadot/cli/internal/auth"
	authcmd "github.com/signadot/cli/internal/command/auth"
)

type GetAuthenticationStatusInput struct{}

type GetAuthenticationStatusOutput struct{}

func getAuthenticationStatus(ctx context.Context, req *mcp.CallToolRequest, in GetAuthenticationStatusInput,
) (*mcp.CallToolResult, GetAuthenticationStatusOutput, error) {
	out := GetAuthenticationStatusOutput{}

	authInfo, err := auth.ResolveAuth()
	if err != nil {
		return nil, out, err
	}

	if !auth.IsAuthenticated(authInfo) {
		return notAuthenticatedResult(), out, nil
	}

	var buf bytes.Buffer
	if err := authcmd.PrintAuthInfo(&buf, authInfo); err != nil {
		return nil, out, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: buf.String()},
		},
	}, out, nil
}
