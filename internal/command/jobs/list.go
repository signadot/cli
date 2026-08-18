package jobs

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/go-sdk/client/jobs"
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

// jobsPaginationAPIVersion pins to the API version at/after which ListJobs
// returns the paginated response ({items, nextCursor, hasMore, totalCount,
// totalPages} instead of a bare array) — see signadot/signadot#7328. The
// CLI's own -o json/-o yaml output stays a plain job array either way
// (below), so this is purely about not depending on the legacy server-side
// path ahead of its 2026-11-16 sunset.
const jobsPaginationAPIVersion = "2026-08-18"

func list(cfg *config.JobList, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	apiVersion := jobsPaginationAPIVersion
	resp, err := cfg.Client.Jobs.ListJobs(
		jobs.NewListJobsParams().WithOrgName(cfg.Org).WithSignadotAPIVersion(&apiVersion),
		nil,
	)
	if err != nil {
		return err
	}
	items := resp.Payload.Items

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
