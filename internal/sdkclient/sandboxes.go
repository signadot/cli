package sdkclient

import (
	"context"
	"fmt"

	"github.com/signadot/go-sdk/client"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
)

const maxSandboxPageSize int64 = 500

// ListAllSandboxes walks the paginated Sandboxes API while keeping the CLI's
// historical plain-array output contract.
func ListAllSandboxes(ctx context.Context, cl *client.SignadotAPI, org string) ([]*models.Sandbox, error) {
	out := make([]*models.Sandbox, 0)
	pageSize := maxSandboxPageSize
	var cursor *string

	for {
		params := sandboxes.NewListSandboxesParams().
			WithDefaults().
			WithContext(ctx).
			WithOrgName(org).
			WithPageSize(&pageSize).
			WithCursor(cursor)

		resp, err := cl.Sandboxes.ListSandboxes(params, nil)
		if err != nil {
			return nil, err
		}
		if resp.Payload == nil {
			return nil, fmt.Errorf("list sandboxes returned empty response")
		}

		out = append(out, resp.Payload.Items...)
		if !resp.Payload.HasMore {
			return out, nil
		}
		if resp.Payload.NextCursor == "" {
			return nil, fmt.Errorf("list sandboxes response had hasMore=true without nextCursor")
		}
		cursor = &resp.Payload.NextCursor
	}
}
