package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type AuthenticatedDummyIn struct {
	Message string `json:"message,omitempty"`
}

type AuthenticatedDummyOut struct {
	Result string `json:"result"`
}

func signadotAuthenticatedDummy(ctx context.Context, req *mcp.CallToolRequest, in AuthenticatedDummyIn,
) (*mcp.CallToolResult, AuthenticatedDummyOut, error) {
	out := AuthenticatedDummyOut{}

	message := in.Message
	if message == "" {
		message = "Hello from authenticated tool!"
	}

	out.Result = "Successfully executed authenticated dummy tool. Message: " + message

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: out.Result},
		},
	}, out, nil
}
