package sandbox

import (
	"errors"
	"fmt"
	"io"

	"github.com/denisbrodbeck/machineid"
	"github.com/signadot/cli/internal/config"
	sbmapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	sbmgr "github.com/signadot/cli/internal/locald/sandboxmanager"
	"github.com/signadot/cli/internal/poll"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/spinner"
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

	if len(req.Spec.Local) > 0 {
		// Confirm sandboxmanager is running and connected to the right cluster
		status, err := sbmgr.GetStatus()
		if err != nil {
			return err
		}
		ciConfig, err := sbmapi.ToCIConfig(status.CiConfig)
		if err != nil {
			return fmt.Errorf("couldn't unmarshal ci-config from sandboxmanager status, %v", err)
		}
		connectErrs := sbmgr.CheckStatusConnectErrors(status, ciConfig)
		if len(connectErrs) != 0 {
			return fmt.Errorf("sandboxmanager is still starting")
		}
		if *req.Spec.Cluster != ciConfig.ConnectionConfig.Cluster {
			return fmt.Errorf("sandbox spec cluster %q does not match connected cluster (%q)",
				*req.Spec.Cluster, ciConfig.ConnectionConfig.Cluster)
		}

		// Set the local machine ID
		machineID, err := machineid.ProtectedID("signadotCLI")
		if err != nil {
			return fmt.Errorf("couldn't read machine-id, %v", err)
		}
		req.Spec.MachineID = machineID
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

	if cfg.Wait {
		// Wait for the sandbox to be ready.
		// store latest resp for output below
		resp, err = waitForReady(cfg, log, resp)
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
		sbURL := cfg.SandboxDashboardURL(resp.RoutingKey)
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

func waitForReady(cfg *config.SandboxApply, out io.Writer, sb *models.Sandbox) (*models.Sandbox, error) {
	fmt.Fprintf(out, "Waiting (up to --wait-timeout=%v) for sandbox to be ready...\n", cfg.WaitTimeout)

	params := sandboxes.NewGetSandboxParams().WithOrgName(cfg.Org).WithSandboxName(sb.Name)

	spin := spinner.Start(out, "Sandbox status")
	defer spin.Stop()

	err := poll.Until(cfg.WaitTimeout, func() bool {
		result, err := cfg.Client.Sandboxes.GetSandbox(params, nil)
		if err != nil {
			// Keep retrying in case it's a transient error.
			spin.Messagef("error: %v", err)
			return false
		}
		sb = result.Payload
		if !sb.Status.Ready {
			spin.Messagef("Not Ready: %s", sb.Status.Message)
			return false
		}
		spin.StopMessagef("Ready: %s", sb.Status.Message)
		return true
	})
	if err != nil {
		spin.StopFail()
		return sb, err
	}
	return sb, nil
}
