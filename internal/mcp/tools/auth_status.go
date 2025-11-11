package tools

import (
	"bytes"
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/signadot/cli/internal/auth"
	authcmd "github.com/signadot/cli/internal/command/auth"
)

type AuthStatusIn struct{}

type AuthStatusOut struct{}

func signadotAuthStatus(ctx context.Context, req *mcp.CallToolRequest, in AuthStatusIn,
) (*mcp.CallToolResult, AuthStatusOut, error) {
	out := AuthStatusOut{}

	authInfo, err := auth.ResolveAuth()
	if err != nil {
		return nil, out, err
	}

	if isAuthenticated(authInfo) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "Not signed in. In a terminal, run:"},
				&mcp.TextContent{Text: "```bash\nsignadot auth login\n```"},
			},
		}, out, nil
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

func isAuthenticated(authInfo *auth.ResolvedAuth) bool {
	return authInfo != nil && (authInfo.ExpiresAt == nil || authInfo.ExpiresAt.After(time.Now()))
}
