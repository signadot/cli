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

func newCreate(sandbox *config.Sandbox) *cobra.Command {
	cfg := &config.SandboxCreate{Sandbox: sandbox}

	cmd := &cobra.Command{
		Use:   "create -f FILENAME",
		Short: "Create sandbox",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return create(cfg, cmd.OutOrStdout())
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func create(cfg *config.SandboxCreate, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	if cfg.Filename == "" {
		return errors.New("must specify sandbox request file with '-f' flag")
	}

	req, err := clio.LoadYAML[models.CreateSandboxRequest](cfg.Filename)
	if err != nil {
		return err
	}

	params := sandboxes.NewCreateNewSandboxParams().WithOrgName(cfg.Org).WithData(req)
	result, err := cfg.Client.Sandboxes.CreateNewSandbox(params, cfg.AuthInfo)
	if err != nil {
		return err
	}
	resp := result.Payload

	fmt.Fprintf(out, "Created sandbox %q (sandbox id: %s) in cluster %q.\n\n",
		req.Name, resp.SandboxID, *req.Cluster)

	// Print warnings, if any.
	for _, msg := range resp.Warnings {
		fmt.Fprintf(out, "WARNING: %s\n\n", msg)
	}

	if cfg.Wait {
		// Wait for the sandbox to be ready.
		fmt.Fprintln(out, "Waiting for sandbox to be ready...")

		params := sandboxes.NewGetSandboxReadyParams().WithOrgName(cfg.Org).WithSandboxID(resp.SandboxID)

		// We use a hot loop because the server implements rate-limiting for us.
		for {
			result, err := cfg.Client.Sandboxes.GetSandboxReady(params, cfg.AuthInfo)
			if err != nil {
				return err
			}
			if result.Payload.Ready {
				break
			}
			// TODO: Show status message when it's added to the API response.
		}

		fmt.Fprintln(out, "Sandbox is ready.")
	}

	return nil
}
