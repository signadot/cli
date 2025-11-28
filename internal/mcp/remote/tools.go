package remote

import (
	"context"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	sbmgr "github.com/signadot/cli/internal/locald/sandboxmanager"
	"github.com/signadot/cli/internal/utils/system"
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
		// Set localMachineID if needed for sandbox tools
		if err := setLocalMachineIDIfNeeded(toolName, in); err != nil {
			return nil, nil, err
		}

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
				if r.session == sess {
					r.session = nil
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

// setLocalMachineIDIfNeeded sets the localMachineID in the spec for create_sandbox
// and update_sandbox tools when local workloads or routing forwards are present.
// It also validates that sandboxmanager is running and connected to the right cluster.
func setLocalMachineIDIfNeeded(toolName string, in ToolInput) error {
	// only apply to create_sandbox and update_sandbox tools
	if toolName != "create_sandbox" && toolName != "update_sandbox" {
		return nil
	}

	// get the spec from the input
	spec, ok := in["spec"].(map[string]any)
	if !ok {
		return nil
	}

	// check if we have local workloads or routing forwards
	var (
		hasLocal           bool
		hasRoutingForwards bool
	)
	if local, ok := spec["local"].([]any); ok && len(local) > 0 {
		hasLocal = true
	}
	if routing, ok := spec["routing"].(map[string]any); ok && routing != nil {
		if forwards, ok := routing["forwards"].([]any); ok && len(forwards) > 0 {
			hasRoutingForwards = true
		}
	}

	if hasLocal || hasRoutingForwards {
		// extract cluster from spec
		var cluster string
		if clusterVal, ok := spec["cluster"]; ok {
			switch v := clusterVal.(type) {
			case string:
				cluster = v
			default:
				return fmt.Errorf("invalid cluster type in spec")
			}
		}
		if cluster == "" {
			return fmt.Errorf("sandbox spec must specify cluster")
		}

		// validate sandboxmanager is running and connected to the right cluster
		_, err := sbmgr.ValidateSandboxManager(&cluster)
		if err != nil {
			return err
		}

		// Set machine ID for local sandboxes
		machineID, err := system.GetMachineID()
		if err != nil {
			return err
		}
		spec["localMachineID"] = machineID
	}

	return nil
}
