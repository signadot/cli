package jobs

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/go-sdk/client/jobs"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newList(job *config.Job) *cobra.Command {
	cfg := &config.JobList{Job: job}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List jobs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return list(cfg, cmd.OutOrStdout())
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func list(cfg *config.JobList, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	// The v2 list endpoint returns a paginated {items,nextCursor,hasMore}
	// envelope (the go-sdk defaults the opt-in to "pagination"). Walk every
	// page so the output is the full list, not just the first page.
	var items []*models.Job
	var cursor *string
	for {
		params := jobs.NewListJobsParams().WithOrgName(cfg.Org)
		if cursor != nil {
			params = params.WithCursor(cursor)
		}
		resp, err := cfg.Client.Jobs.ListJobs(params, nil)
		if err != nil {
			return err
		}
		items = append(items, resp.Payload.Items...)
		if !resp.Payload.HasMore || resp.Payload.NextCursor == "" {
			break
		}
		next := resp.Payload.NextCursor
		cursor = &next
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printJobTable(cfg, out, items)
	case config.OutputFormatJSON:
		return print.RawJSON(out, items)
	case config.OutputFormatYAML:
		return print.RawYAML(out, items)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
