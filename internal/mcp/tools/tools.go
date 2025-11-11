package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/signadot/cli/internal/config"
)

func Setup(cfg *config.MCPRun, srv *mcp.Server) {
	mcp.AddTool(srv, &mcp.Tool{
		Name: "signadotAuthStatus",
		Description: `<usecase>
Checks Signadot authentication status.
</usecase>
<instructions>
Use this tool to verify authentication when other Signadot tools are unavailable or not found.
</instructions>`,
	}, signadotAuthStatus)
}
