package jobs

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/spf13/cobra"
)

func newGet(job *config.Job) *cobra.Command {
	cfg := &config.JobGet{Job: job}

	cmd := &cobra.Command{
		Use:   "get NAME",
		Short: "Get job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return get(cfg, cmd.OutOrStdout(), args[0])
		},
	}

	return cmd
}

func get(cfg *config.JobGet, out io.Writer, name string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	job, err := getJob(cfg.Job, name)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printJobDetails(cfg, out, job)
	case config.OutputFormatJSON:
		return print.RawJSON(out, job)
	case config.OutputFormatYAML:
		return print.RawYAML(out, job)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
