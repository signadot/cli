package sdkclient

import (
	"context"
	"fmt"

	"github.com/signadot/go-sdk/client"
	"github.com/signadot/go-sdk/client/jobs"
	"github.com/signadot/go-sdk/models"
)

const maxJobPageSize int64 = 500

// ListAllJobs walks the paginated Jobs API while keeping the CLI's historical
// plain-array output contract.
func ListAllJobs(ctx context.Context, cl *client.SignadotAPI, org string) ([]*models.Job, error) {
	out := make([]*models.Job, 0)
	pageSize := maxJobPageSize
	var cursor *string

	for {
		params := jobs.NewListJobsParams().
			WithDefaults().
			WithContext(ctx).
			WithOrgName(org).
			WithPageSize(&pageSize).
			WithCursor(cursor)

		resp, err := cl.Jobs.ListJobs(params, nil)
		if err != nil {
			return nil, err
		}
		if resp.Payload == nil {
			return nil, fmt.Errorf("list jobs returned empty response")
		}

		out = append(out, resp.Payload.Items...)
		if !resp.Payload.HasMore {
			return out, nil
		}
		if resp.Payload.NextCursor == "" {
			return nil, fmt.Errorf("list jobs response had hasMore=true without nextCursor")
		}
		cursor = &resp.Payload.NextCursor
	}
}
