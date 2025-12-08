package tools

import (
	"context"
	"log/slog"
	"reflect"
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

	registeredTools map[string]*mcp.Tool // Track registered remote tools by name
	lastAuthStatus  bool                 // Track last authentication status
}

func NewTools(log *slog.Logger, mcpSrv *mcp.Server, remoteMgr *remote.Remote) *Tools {
	return &Tools{
		log:       log.With("component", "tools"),
		mcpSrv:    mcpSrv,
		remoteMgr: remoteMgr,

		registeredTools: map[string]*mcp.Tool{},
		lastAuthStatus:  auth.IsAuthenticated(nil),
	}
}

func (t *Tools) Setup() {
	// Always register public tools
	mcp.AddTool(t.mcpSrv, &mcp.Tool{
		Name: "get_authentication_status",
		Description: `<usecase>
Checks the current authentication status with the Signadot Control Plane.
</usecase>
<instructions>
This tool can be used to troubleshoot authentication issues or check the current authentication status.
When requested, ask the user to run the provided command in a terminal to authenticate.
</instructions>`,
	}, getAuthenticationStatus)
}

func (t *Tools) Update(ctx context.Context, meta *remote.Meta) {
	if meta == nil {
		t.log.Warn("remote metadata is not available")
		return
	}

	t.Lock()
	defer t.Unlock()

	t.log.Debug("updating tools")

	isAuthenticated := auth.IsAuthenticated(nil)
	authStatusChanged := isAuthenticated != t.lastAuthStatus
	t.lastAuthStatus = isAuthenticated

	// Build a map of new tools by name
	newTools := make(map[string]*mcp.Tool)
	for _, tool := range meta.Tools {
		newTools[tool.Name] = tool
	}

	// Find tools to remove (in registered but not in new)
	var toolsToRemove []string
	for name := range t.registeredTools {
		if _, exists := newTools[name]; !exists {
			toolsToRemove = append(toolsToRemove, name)
		}
	}

	// Find tools to add or update (in new but not in registered, or changed, or
	// auth status changed)
	var toolsToAddOrUpdate []*mcp.Tool
	for name, tool := range newTools {
		registered, exists := t.registeredTools[name]
		if !exists {
			// New tool
			toolsToAddOrUpdate = append(toolsToAddOrUpdate, tool)
		} else if authStatusChanged || !toolEqual(registered, tool) {
			// Tool changed or auth status changed (which requires handler update)
			toolsToAddOrUpdate = append(toolsToAddOrUpdate, tool)
		}
	}

	// Remove tools that are no longer present
	if len(toolsToRemove) > 0 {
		t.log.Debug("removing tools", "tools", toolsToRemove)
		t.mcpSrv.RemoveTools(toolsToRemove...)
		for _, name := range toolsToRemove {
			delete(t.registeredTools, name)
		}
	}

	// Add or update tools
	for _, tool := range toolsToAddOrUpdate {
		t.log.Debug("registering tool", "tool", tool.Name)
		// Create a copy of the tool to avoid modifying the original
		toolCopy := *tool
		if isAuthenticated {
			// register tool with the remote proxy handler
			mcp.AddTool(t.mcpSrv, &toolCopy, t.remoteMgr.ToolHandler(tool.Name))
		} else {
			// register tool with the local auth request handler (and override
			// the output schema)
			toolCopy.OutputSchema = nil
			mcp.AddTool(t.mcpSrv, &toolCopy, t.localAuthRequestHandler(tool.Name))
		}
		// Update our tracking - store a copy of the original tool (not the modified one)
		// so we can properly compare on future updates
		originalCopy := *tool
		t.registeredTools[tool.Name] = &originalCopy
	}
}

// toolEqual compares two tools to determine if they're functionally equivalent.
// It compares the tool definition (excluding handler, which we track separately).
func toolEqual(a, b *mcp.Tool) bool {
	if a.Name != b.Name {
		return false
	}
	if a.Description != b.Description {
		return false
	}
	if !reflect.DeepEqual(a.InputSchema, b.InputSchema) {
		return false
	}
	if !reflect.DeepEqual(a.OutputSchema, b.OutputSchema) {
		return false
	}
	return true
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
