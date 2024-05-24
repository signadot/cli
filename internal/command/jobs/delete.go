package jobs

//
//import (
//	"errors"
//	"fmt"
//	"io"
//	"strings"
//
//	"github.com/signadot/cli/internal/config"
//	"github.com/signadot/cli/internal/poll"
//	"github.com/signadot/cli/internal/spinner"
//	jobs "github.com/signadot/go-sdk/client/route_groups"
//	"github.com/spf13/cobra"
//)
//
//func newDelete(job *config.Job) *cobra.Command {
//	cfg := &config.JobDelete{Job: job}
//
//	cmd := &cobra.Command{
//		Use:   "delete { NAME | -f FILENAME [ --set var1=val1 --set var2=val2 ... ] }",
//		Short: "Delete job",
//		Args:  cobra.MaximumNArgs(1),
//		RunE: func(cmd *cobra.Command, args []string) error {
//			return rgDelete(cfg, cmd.ErrOrStderr(), args)
//		},
//	}
//
//	cfg.AddFlags(cmd)
//
//	return cmd
//}
//
//func rgDelete(cfg *config.JobDelete, log io.Writer, args []string) error {
//	if err := cfg.InitAPIConfig(); err != nil {
//		return err
//	}
//	// Get the name either from a file or from the command line.
//	var name string
//	if cfg.Filename == "" {
//		if len(args) == 0 {
//			return errors.New("must specify filename (-f) or job name")
//		}
//		if len(cfg.TemplateVals) != 0 {
//			return errors.New("must specify filename (-f) to use --set")
//		}
//		name = args[0]
//	} else {
//		if len(args) != 0 {
//			return errors.New("must not provide args when filename (-f) specified")
//		}
//		rg, err := loadJob(cfg.Filename, cfg.TemplateVals, true /* forDelete */)
//		if err != nil {
//			return err
//		}
//		name = rg.Name
//	}
//
//	if name == "" {
//		return errors.New("job name is required")
//	}
//
//	// Delete the job.
//	params := jobs.NewDeleteRoutegroupParams().
//		WithOrgName(cfg.Org).
//		WithRoutegroupName(name)
//	_, err := cfg.Client.Jobs.DeleteRoutegroup(params, nil)
//	if err != nil {
//		return err
//	}
//
//	fmt.Fprintf(log, "Deleted job %q.\n\n", name)
//
//	if cfg.Wait {
//		// Wait for the API server to completely reflect deletion.
//		if err := waitForDeleted(cfg, log, name); err != nil {
//			fmt.Fprintf(log, "\nDeletion was initiated, but the job may still exist in a terminating state. To check status, run:\n\n")
//			fmt.Fprintf(log, "  signadot job get %v\n\n", name)
//			return err
//		}
//	}
//
//	return nil
//}
//
//func waitForDeleted(cfg *config.JobDelete, log io.Writer, routegroupName string) error {
//	fmt.Fprintf(log, "Waiting (up to --wait-timeout=%v) for job to finish terminating...\n", cfg.WaitTimeout)
//
//	params := jobs.NewGetRoutegroupParams().WithOrgName(cfg.Org).WithRoutegroupName(routegroupName)
//
//	spin := spinner.Start(log, "Job status")
//	defer spin.Stop()
//
//	err := poll.Until(cfg.WaitTimeout, func() bool {
//		result, err := cfg.Client.Jobs.GetRoutegroup(params, nil)
//		if err != nil {
//			// If it's a "not found" error, that's what we wanted.
//			// TODO: Pass through an error code so we don't have to rely on the error message.
//			if strings.Contains(err.Error(), "unable to fetch job: not found") {
//				spin.StopMessage("Terminated")
//				return true
//			}
//
//			// Otherwise, keep retrying in case it's a transient error.
//			spin.Messagef("error: %v", err)
//			return false
//		}
//		status := result.Payload.Status
//		if status.Ready {
//			spin.Message("Waiting for job to terminate")
//			return false
//		}
//		spin.Messagef("%s: %s", status.Reason, status.Message)
//		return false
//	})
//	if err != nil {
//		spin.StopFail()
//		return err
//	}
//	return nil
//}
