package sandbox

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/signadot/cli/internal/clio"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/poll"
	"github.com/signadot/cli/internal/spinner"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newDelete(sandbox *config.Sandbox) *cobra.Command {
	cfg := &config.SandboxDelete{Sandbox: sandbox}

	cmd := &cobra.Command{
		Use:   "delete { -f FILENAME | NAME }",
		Short: "Delete sandbox",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return delete(cfg, cmd.ErrOrStderr(), args)
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func delete(cfg *config.SandboxDelete, log io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	if cfg.Filename == "" && len(args) == 0 {
		return errors.New("must specify either filename or sandbox name")
	}
	if cfg.Filename != "" && len(args) > 0 {
		return errors.New("can't specify both filename and sandbox name")
	}

	// Get the name either from a file or from the command line.
	var name string
	if len(args) > 0 {
		name = args[0]
	} else {
		req, err := clio.LoadYAML[models.Sandbox](cfg.Filename)
		if err != nil {
			return err
		}
		name = req.Name
	}
	if name == "" {
		return errors.New("sandbox name is required")
	}

	// Delete the sandbox.
	params := sandboxes.NewDeleteSandboxParams().WithOrgName(cfg.Org).WithSandboxName(name)
	_, err := cfg.Client.Sandboxes.DeleteSandbox(params, nil)
	if err != nil {
		return err
	}

	fmt.Fprintf(log, "Deleted sandbox %q.\n\n", name)

	if cfg.Wait {
		// Wait for the API server to completely reflect deletion.
		if err := waitForDeleted(cfg, log, name); err != nil {
			fmt.Fprintf(log, "\nDeletion was initiated, but the sandbox may still exist in a terminating state. To check status, run:\n\n")
			fmt.Fprintf(log, "  signadot sandbox get-status %v\n\n", name)
			return err
		}
	}

	return nil
}

func waitForDeleted(cfg *config.SandboxDelete, log io.Writer, sandboxName string) error {
	fmt.Fprintf(log, "Waiting (up to --wait-timeout=%v) for sandbox to finish terminating...\n", cfg.WaitTimeout)

	params := sandboxes.NewGetSandboxParams().WithOrgName(cfg.Org).WithSandboxName(sandboxName)

	spin := spinner.Start(log, "Sandbox status")
	defer spin.Stop()

	err := poll.Until(cfg.WaitTimeout, func() bool {
		result, err := cfg.Client.Sandboxes.GetSandbox(params, nil)
		if err != nil {
			// If it's a "not found" error, that's what we wanted.
			// TODO: Pass through an error code so we don't have to rely on the error message.
			if strings.Contains(err.Error(), "can't get sandbox: not found") {
				spin.StopMessage("Terminated")
				return true
			}

			// Otherwise, keep retrying in case it's a transient error.
			spin.Messagef("error: %v", err)
			return false
		}
		status := result.Payload.Status
		if status.Ready {
			spin.Message("Waiting for sandbox to terminate")
			return false
		}
		spin.Messagef("%s: %s", status.Reason, status.Message)
		return false
	})
	if err != nil {
		spin.StopFail()
		return err
	}
	return nil
}
