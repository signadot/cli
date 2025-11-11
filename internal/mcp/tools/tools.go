package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/signadot/cli/internal/auth"
	"github.com/signadot/cli/internal/config"
)

const (
	authenticatedDummyToolName = "signadotAuthenticatedDummy"
)

var (
	// authenticatedTools is the list of tool names that require authentication
	authenticatedTools = []string{authenticatedDummyToolName}
)

// Setup registers all tools. Public tools are always registered,
// authenticated tools are only registered if the user is authenticated.
func Setup(cfg *config.MCPRun, srv *mcp.Server) {
	// Always register public tools
	mcp.AddTool(srv, &mcp.Tool{
		Name: "signadotAuthStatus",
		Description: `<usecase>
Checks Signadot authentication status.
</usecase>
<instructions>
Use this tool to verify authentication when other Signadot tools are unavailable or not found.
</instructions>`,
	}, signadotAuthStatus)

	// Register authenticated tools based on current auth status
	Update(srv)
}

// Update checks the current authentication status and adds or removes
// authenticated tools accordingly. The MCP SDK will automatically send
// tools/list_changed notifications to clients when tools are added/removed.
func Update(srv *mcp.Server) {
	if auth.IsAuthenticated(nil) {
		// User is authenticated, ensure authenticated tools are registered
		ensureAuthenticatedToolsRegistered(srv)
	} else {
		// User is not authenticated, remove authenticated tools
		removeAuthenticatedTools(srv)
	}
}

// ensureAuthenticatedToolsRegistered adds authenticated tools if they're not
// already registered. Note: The MCP SDK may handle duplicate registrations
// gracefully, but we try to avoid them.
func ensureAuthenticatedToolsRegistered(srv *mcp.Server) {
	// Register the authenticated dummy tool
	mcp.AddTool(srv, &mcp.Tool{
		Name: authenticatedDummyToolName,
		Description: `<usecase>
A dummy tool that requires authentication to be available.
</usecase>
<instructions>
This is an example authenticated tool. It will only appear in the tool list when you are authenticated with Signadot.
</instructions>`,
	}, signadotAuthenticatedDummy)
}

// removeAuthenticatedTools removes all authenticated tools from the server.
// This will trigger tools/list_changed notifications to all connected clients.
func removeAuthenticatedTools(srv *mcp.Server) {
	srv.RemoveTools(authenticatedTools...)
}
