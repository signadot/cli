package jobs

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/go-sdk/client/jobs"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newSubmit(job *config.Job) *cobra.Command {
	cfg := &config.JobSubmit{Job: job}

	cmd := &cobra.Command{
		Use:   "submit -f FILENAME [ --set var1=val1 --set var2=val2 ... ]",
		Short: "Submit a job with variable expansion",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return submit(cmd.Context(), cfg, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}

	cfg.AddFlags(cmd)
	return cmd
}

func submit(ctx context.Context, cfg *config.JobSubmit, outW, errW io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	if cfg.Filename == "" {
		return errors.New("must specify job request file with '-f' flag")
	}
	req, err := loadJob(cfg.Filename, cfg.TemplateVals, false /*forDelete */)
	if err != nil {
		return err
	}

	params := jobs.NewCreateJobParams().
		WithOrgName(cfg.Org).WithData(req)
	result, err := cfg.Client.Jobs.CreateJob(params, nil)
	if err != nil {
		return err
	}
	resp := result.Payload

	return writeOutput(ctx, cfg, outW, errW, resp)
}

func writeOutput(ctx context.Context, cfg *config.JobSubmit, outW, errW io.Writer, resp *models.Job) error {
	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		// Print info on how to access the job.
		fmt.Fprintf(outW, "Job %s queued on Job Runner Group: %s\n", resp.Name, resp.Spec.RunnerGroup)
		fmt.Fprintf(outW, "\nDashboard page: %v\n\n", cfg.JobDashboardUrl(resp.Name))

		var err error
		if cfg.Attach {
			err = waitForJob(ctx, cfg, outW, errW, resp.Name)
		}
		return err
	case config.OutputFormatJSON:
		return print.RawJSON(outW, resp)
	case config.OutputFormatYAML:
		return print.RawYAML(outW, resp)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
