package remote

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolInput is a generic input type for proxied tools that maps parameter names
// to their values.
type ToolInput map[string]any

// ToolOutput is a generic output type for proxied tools that maps output field
// names to their values.
type ToolOutput map[string]any

// ToolHandler returns a handler function that proxies tool calls to the remote
// MCP server. The returned handler uses the Remote's managed session to forward
// tool invocations to the remote server and returns the structured output from
// the remote tool.
func (r *Remote) ToolHandler(toolName string) func(ctx context.Context, req *mcp.CallToolRequest, in ToolInput) (*mcp.CallToolResult, ToolOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in ToolInput) (*mcp.CallToolResult, ToolOutput, error) {
		// Get or create a remote session
		sess, err := r.Session()
		if err != nil {
			return nil, nil, err
		}

		// Convert input to arguments map for the remote tool call
		arguments := map[string]any(in)

		// Prepare parameters for the remote tool invocation
		params := &mcp.CallToolParams{
			Name:      toolName,
			Arguments: arguments,
		}

		// Call the remote tool using the session
		result, err := sess.CallTool(ctx, params)
		if err != nil {
			// If the connection was closed (e.g., by KeepAlive), recreate the
			// session and retry once
			if errors.Is(err, mcp.ErrConnectionClosed) {
				// Clear the session so it will be recreated on next call
				r.mu.Lock()
				if r.remoteSession == sess {
					r.remoteSession = nil
				}
				r.mu.Unlock()

				// Get a new session and retry
				sess, err = r.Session()
				if err != nil {
					return nil, nil, err
				}
				result, err = sess.CallTool(ctx, params)
				if err != nil {
					return nil, nil, err
				}
			} else {
				return nil, nil, err
			}
		}
		if result.IsError {
			return nil, nil, errors.New(result.Content[0].(*mcp.TextContent).Text)
		}

		// Extract structured content from the remote tool result
		var output ToolOutput
		if result != nil && result.StructuredContent != nil {
			// Convert StructuredContent to map[string]interface{} if possible
			if contentMap, ok := result.StructuredContent.(map[string]any); ok {
				output = ToolOutput(contentMap)
			} else {
				// If StructuredContent is not a map, return empty output
				// The tool schema validation will handle this appropriately
				output = ToolOutput{}
			}
		} else {
			output = ToolOutput{}
		}

		// Return the result with the extracted structured output
		return result, output, nil
	}
}
