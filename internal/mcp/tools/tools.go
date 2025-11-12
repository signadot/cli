package tools

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/signadot/cli/internal/auth"
	"github.com/signadot/cli/internal/mcp/remote"
)

type Tools struct {
	sync.Mutex
	log       *slog.Logger
	mcpSrv    *mcp.Server
	remoteMgr *remote.Remote
}

func NewTools(log *slog.Logger, mcpSrv *mcp.Server, remoteMgr *remote.Remote) *Tools {
	return &Tools{
		log:       log.With("component", "tools"),
		mcpSrv:    mcpSrv,
		remoteMgr: remoteMgr,
	}
}

func (t *Tools) Setup() {
	// Always register public tools
	mcp.AddTool(t.mcpSrv, &mcp.Tool{
		Name: "signadotAuthStatus",
		Description: `<usecase>
Checks Signadot authentication status.
</usecase>
<instructions>
Use this tool to verify authentication when other Signadot tools are unavailable or not found.
</instructions>`,
	}, signadotAuthStatus)
}

func (t *Tools) Update(ctx context.Context, meta *remote.Meta) error {
	if meta == nil {
		return fmt.Errorf("remote metadata is not available")
	}

	// register remote tools
	for _, tool := range meta.Tools {
		if auth.IsAuthenticated(nil) {
			// register tool with the remote proxy handler
			mcp.AddTool(t.mcpSrv, tool, t.remoteMgr.ToolHandler(tool.Name))
		} else {
			// register tool with the local auth request handler (and override
			// the output schema)
			tool.OutputSchema = nil
			mcp.AddTool(t.mcpSrv, tool, t.localAuthRequestHandler(tool.Name))
		}
	}
	return nil
}

func (t *Tools) localAuthRequestHandler(_ string) func(ctx context.Context, req *mcp.CallToolRequest, in remote.ToolInput) (*mcp.CallToolResult, remote.ToolOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in remote.ToolInput) (*mcp.CallToolResult, remote.ToolOutput, error) {
		out := remote.ToolOutput{}
		return notAuthenticatedResult(), out, nil
	}
}

func notAuthenticatedResult() *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Not authenticated. In a terminal, run:"},
			&mcp.TextContent{Text: "```bash\nsignadot auth login\n```"},
		},
	}
}
