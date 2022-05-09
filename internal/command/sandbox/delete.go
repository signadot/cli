package sandbox

import (
	"errors"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/clio"
	"github.com/signadot/cli/internal/config"
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
			return delete(cfg, cmd.OutOrStdout(), args)
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func delete(cfg *config.SandboxDelete, out io.Writer, args []string) error {
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
		req, err := clio.LoadYAML[models.CreateSandboxRequest](cfg.Filename)
		if err != nil {
			return err
		}
		name = req.Name
	}
	if name == "" {
		return errors.New("sandbox name is required")
	}

	// List sandboxes to find the one with the desired name.
	// TODO: Use GetSandboxByName when it's available.
	resp, err := cfg.Client.Sandboxes.GetSandboxes(sandboxes.NewGetSandboxesParams().WithOrgName(cfg.Org), cfg.AuthInfo)
	if err != nil {
		return err
	}
	var id string
	for _, sb := range resp.Payload.Sandboxes {
		if sb.Name == name {
			id = sb.ID
			break
		}
	}
	if id == "" {
		return fmt.Errorf("Sandbox %q not found", name)
	}

	// Delete the sandbox.
	params := sandboxes.NewDeleteSandboxByIDParams().WithOrgName(cfg.Org).WithSandboxID(id)
	_, err = cfg.Client.Sandboxes.DeleteSandboxByID(params, cfg.AuthInfo)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "Deleted sandbox %q.\n", name)

	return nil
}
