package sandbox

import (
	"errors"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/builder"
	"github.com/signadot/cli/internal/config"
	sbmapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	sbmgr "github.com/signadot/cli/internal/locald/sandboxmanager"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"

	"github.com/spf13/cobra"
)

func newApply(sandbox *config.Sandbox) *cobra.Command {
	cfg := &config.SandboxApply{Sandbox: sandbox}

	cmd := &cobra.Command{
		Use:   "apply -f FILENAME [ --set var1=val1 --set var2=val2 ... ]",
		Short: "Create or update a sandbox with variable expansion",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return apply(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}
	cfg.AddFlags(cmd)
	return cmd
}

func apply(cfg *config.SandboxApply, out, log io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	if cfg.Filename == "" {
		return errors.New("must specify sandbox request file with '-f' flag")
	}

	// Load the sandbox spec
	req, err := loadSandbox(cfg.Filename, cfg.TemplateVals, false /*forDelete */)
	if err != nil {
		return err
	}
	if req.Spec.Cluster == nil {
		return fmt.Errorf("sandbox spec must specify cluster")
	}

	var status *sbmapi.StatusResponse
	if len(req.Spec.Local) > 0 || req.Spec.Routing != nil && len(req.Spec.Routing.Forwards) > 0 {
		// Validate sandboxmanager is running and connected to the right cluster
		status, err = sbmgr.ValidateSandboxManager(req.Spec.Cluster)
		if err != nil {
			return err
		}

		// Set machine ID for local sandboxes
		sb, err := builder.
			BuildSandbox(req.Name, builder.WithData(*req)).
			SetMachineID().
			Build()
		if err != nil {
			return err
		}
		req = &sb
	}

	// Send the request to the SaaS
	params := sandboxes.NewApplySandboxParams().
		WithOrgName(cfg.Org).WithSandboxName(req.Name).WithData(req)
	result, err := cfg.Client.Sandboxes.ApplySandbox(params, nil)
	if err != nil {
		return err
	}
	resp := result.Payload

	fmt.Fprintf(log, "Created sandbox %q (routing key: %s) in cluster %q.\n\n",
		req.Name, resp.RoutingKey, *req.Spec.Cluster)

	if len(req.Spec.Local) > 0 && status.OperatorInfo == nil {
		// we are dealing with an old operator that doesn't support sandboxes
		// watcher, go ahead and register the sandbox in sandboxmanager.
		if err = sbmgr.RegisterSandbox(resp.Name, resp.RoutingKey); err != nil {
			return fmt.Errorf("couldn't register sandbox in sandboxmanager, %v", err)
		}
	}

	if cfg.Wait {
		// Wait for the sandbox to be ready.
		// store latest resp for output below
		resp, err = utils.WaitForSandboxReady(cfg.API, log, resp.Name, cfg.WaitTimeout)
		if err != nil {
			writeOutput(cfg, out, resp)
			fmt.Fprintf(log, "\nThe sandbox was applied, but it may not be ready yet. To check status, run:\n\n")
			fmt.Fprintf(log, "  signadot sandbox get %v\n\n", req.Name)
			return err
		}
		writeOutput(cfg, out, resp)
		fmt.Fprintf(log, "\nThe sandbox %q was applied and is ready.\n", resp.Name)
		return nil
	}
	return writeOutput(cfg, out, resp)
}

func writeOutput(cfg *config.SandboxApply, out io.Writer, resp *models.Sandbox) error {
	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		// Print info on how to access the sandbox.
		sbURL := cfg.SandboxDashboardURL(resp.Name)
		fmt.Fprintf(out, "\nDashboard page: %v\n\n", sbURL)

		if len(resp.Endpoints) > 0 {
			if err := printEndpointTable(out, resp.Endpoints); err != nil {
				return err
			}
		}
		return nil
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
