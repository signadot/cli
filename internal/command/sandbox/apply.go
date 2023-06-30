package sandbox

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/local"
	sbmgr "github.com/signadot/cli/internal/locald/sandboxmanager"
	"github.com/signadot/cli/internal/poll"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/spinner"
	"github.com/signadot/cli/internal/utils/system"
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
	req, err := loadSandbox(cfg.Filename, cfg.TemplateVals, false /*forDelete */)
	if err != nil {
		return err
	}
	signadotDir, err := system.GetSignadotDir()
	if err != nil {
		return err
	}

	// TODO test if local is up
	var resp *models.Sandbox
	hasLocal, err := local.IsLocked(signadotDir)
	if err != nil {
		return err
	}
	if hasLocal {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resp, err = sbmgr.Apply(ctx, cfg.Org, req.Name, req.Spec)
		if err != nil {
			return err
		}

	} else {
		params := sandboxes.NewApplySandboxParams().
			WithOrgName(cfg.Org).WithSandboxName(req.Name).WithData(req)
		result, err := cfg.Client.Sandboxes.ApplySandbox(params, nil)
		if err != nil {
			return err
		}
		resp = result.Payload
	}

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
