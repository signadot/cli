package jobs

import (
	"fmt"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/go-sdk/client/jobs"
	"github.com/spf13/cobra"
	"io"
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
	params := jobs.NewGetJobParams().WithOrgName(cfg.Org).WithJobName(name)
	resp, err := cfg.Client.Jobs.GetJob(params, nil)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printJobDetails(cfg.Job, out, resp.Payload)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
