package sandbox

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/signadot/cli/internal/builder"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/devbox"
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
		Long: `Create or update a sandbox with variable expansion.

--dry-run=client renders the spec and validates it locally, printing the result
instead of applying it, and needs no API credentials. The output is a spec, so it
can be reviewed, diffed, and passed straight back to -f.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return apply(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}
	cfg.AddFlags(cmd)
	return cmd
}

func apply(cfg *config.SandboxApply, out, log io.Writer, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()

	if cfg.Filename == "" {
		return errors.New("must specify sandbox request file with '-f' flag")
	}
	if cfg.DryRun == config.DryRunServer {
		return errors.New("--dry-run=server is not available yet; " +
			"use --dry-run=client to render and validate locally")
	}

	// Render before authenticating, so that --dry-run=client works with no
	// credentials at all and a spec that cannot be rendered fails the same way
	// whether or not the caller is logged in.
	doc, err := utils.LoadUnstructuredTemplate(cfg.Filename, cfg.TemplateVals, false /* forDelete */)
	if err != nil {
		return err
	}
	req, err := unstructuredToSandbox(doc)
	if err != nil {
		return err
	}
	if req.Spec.Cluster == nil {
		return fmt.Errorf("sandbox spec must specify cluster")
	}
	if cfg.DryRun == config.DryRunClient {
		return writeRenderedSpec(cfg, out, doc)
	}

	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	var status *sbmapi.StatusResponse
	if len(req.Spec.Local) > 0 || req.Spec.Routing != nil && len(req.Spec.Routing.Forwards) > 0 {
		// Validate sandboxmanager is running and connected to the right cluster
		status, err = sbmgr.ValidateSandboxManager(req.Spec.Cluster)
		if err != nil {
			return err
		}

		// Set devbox ID for local sandboxes
		sb, err := builder.
			BuildSandbox(req.Name, builder.WithData(*req)).
			SetDevboxID(status.DevboxSession.DevboxId).
			Build()
		if err != nil {
			return err
		}
		req = &sb
	}

	// Send the request to the SaaS
	params := sandboxes.NewApplySandboxParams().
		WithContext(ctx).
		WithOrgName(cfg.Org).
		WithSandboxName(req.Name).
		WithData(req)
	result, err := cfg.Client.Sandboxes.ApplySandbox(params, nil)
	if err != nil {
		return err
	}
	resp := result.Payload

	fmt.Fprintf(log, "Created sandbox %q (routing key: %s) in cluster %q.\n\n",
		req.Name, resp.RoutingKey, *req.Spec.Cluster)

	if len(req.Spec.Local) > 0 && status.OperatorInfo == nil {
		id, err := devbox.GetID(ctx, cfg.API, false, "" /* name set on connect */)
		if err != nil {
			return err
		}
		_ = id
	}

	if cfg.Wait {
		// Wait for the sandbox to be ready.
		// store latest resp for output below
		resp, err = utils.WaitForSandboxReady(ctx, cfg.API, log, resp.Name, cfg.WaitTimeout)
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

// writeRenderedSpec prints the rendered spec for --dry-run. YAML is the default
// because the output's job is to be read, diffed, and fed back to -f.
func writeRenderedSpec(cfg *config.SandboxApply, out io.Writer, doc any) error {
	switch cfg.OutputFormat {
	case config.OutputFormatDefault, config.OutputFormatYAML:
		return print.RawYAML(out, doc)
	case config.OutputFormatJSON:
		return print.RawJSON(out, doc)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
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
