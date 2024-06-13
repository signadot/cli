package jobs

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/go-sdk/client/artifacts"
	"github.com/signadot/go-sdk/client/jobs"
	"github.com/signadot/go-sdk/models"
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
	artifacts, err := getArtifacts(cfg.Job, job)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printJobDetails(cfg.Job, out, job, artifacts)
	case config.OutputFormatJSON:
		return printRawJob(out, print.RawJSON, job, artifacts)
	case config.OutputFormatYAML:
		return printRawJob(out, print.RawK8SYAML, job, artifacts)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}

func getJob(cfg *config.Job, jobName string) (*models.Job, error) {
	params := jobs.NewGetJobParams().WithOrgName(cfg.Org).WithJobName(jobName)
	resp, err := cfg.Client.Jobs.GetJob(params, nil)
	if err != nil {
		return nil, err
	}

	return resp.Payload, nil
}

func getArtifacts(cfg *config.Job, job *models.Job) ([]*models.JobArtifact, error) {
	params := artifacts.NewListJobAttemptArtifactsParams().
		WithOrgName(cfg.Org).
		WithJobAttempt(job.Status.Attempts[0].ID).
		WithJobName(job.Name)

	resp, err := cfg.Client.Artifacts.ListJobAttemptArtifacts(params, nil)
	if err != nil {
		return []*models.JobArtifact{}, nil
	}

	return resp.Payload, nil
}
