package jobs

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/client/jobs"
	"github.com/spf13/cobra"
)

func newCancel(job *config.Job) *cobra.Command {
	cfg := &config.JobDelete{Job: job}

	cmd := &cobra.Command{
		Use:   "cancel NAME }",
		Short: "Cancel job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return jobCancel(cfg, cmd.ErrOrStderr(), args[0])
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func jobCancel(cfg *config.JobDelete, log io.Writer, name string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	// Cancel the job.
	params := jobs.NewCancelJobParams().
		WithOrgName(cfg.Org).
		WithJobName(name)
	_, err := cfg.Client.Jobs.CancelJob(params, nil)
	if err != nil {
		return err
	}

	fmt.Fprintf(log, "Job %q canceled.\n\n", name)

	return nil
}
