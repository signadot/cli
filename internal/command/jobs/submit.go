package jobs

import (
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
		Short: "Create or update a job with variable expansion",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return submit(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func submit(cfg *config.JobSubmit, out, log io.Writer, args []string) error {
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

	fmt.Fprintf(log, "Job submitted as %q on Job Runner Group: %s", resp.Name, resp.Spec.RunnerGroup)

	return writeOutput(cfg, out, resp)
}

func writeOutput(cfg *config.JobSubmit, out io.Writer, resp *models.Job) error {
	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		// Print info on how to access the job.
		fmt.Fprintf(out, "\nDashboard page: %v\n\n", cfg.JobDashboardUrl(resp.Name))

		return nil
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
