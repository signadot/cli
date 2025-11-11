package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/signadot/cli/internal/auth"
)

// CreateAuthCheckMiddleware creates middleware that checks authentication
// status when list_tools is called, ensuring tools are up-to-date for each session.
func CreateAuthCheckMiddleware(monitor *auth.Monitor) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			// Check authentication before handling list_tools requests
			// This ensures tools are up-to-date for each session
			if method == "tools/list" {
				monitor.Check()
			}
			return next(ctx, method, req)
		}
	}
}
