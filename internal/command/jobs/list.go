package jobs

import (
	"context"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/sdkclient"
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
	jobs, err := sdkclient.ListAllJobs(context.Background(), cfg.Client, cfg.Org)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printJobTable(cfg, out, jobs)
	case config.OutputFormatJSON:
		return print.RawJSON(out, jobs)
	case config.OutputFormatYAML:
		return print.RawYAML(out, jobs)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
