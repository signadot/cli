package sandbox

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	sandbox_ui "github.com/signadot/cli/internal/ui/sandbox"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newCreate(sandbox *config.Sandbox) *cobra.Command {
	cfg := &config.SandboxCreate{Sandbox: sandbox}

	cmd := &cobra.Command{
		Use:   "create --kubernetes-workload=KIND/NAMESPACE/NAME [--ttl=DURATION]",
		Short: "Create a sandbox from a Kubernetes workload",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return create(cmd.Context(), cfg, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	cfg.AddFlags(cmd)
	return cmd
}

func create(ctx context.Context, cfg *config.SandboxCreate, out, log io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	sandbox_ui.Run(ctx, cfg)

	// Parse the kubernetes workload
	kind, namespace, name, err := parseKubernetesWorkload(cfg.KubernetesWorkload)
	if err != nil {
		return fmt.Errorf("invalid kubernetes workload format: %v", err)
	}

	// Generate sandbox name
	sandboxName := generateSandboxName(name)

	// Create sandbox spec
	sandboxSpec := &models.Sandbox{
		Name: sandboxName,
		Spec: &models.SandboxSpec{
			Cluster: &cfg.Cluster,
			Forks: []*models.SandboxFork{
				{

					ForkOf: &models.SandboxForkOf{
						Kind:      &kind,
						Name:      &name,
						Namespace: &namespace,
					},
				},
			},
		},
	}

	// Add TTL if specified
	if cfg.TTL != "" {
		sandboxSpec.Spec.TTL = &models.SandboxTTL{
			Duration: cfg.TTL,
		}
	}

	// Send the request to the API
	params := sandboxes.NewApplySandboxParams().
		WithOrgName(cfg.Org).WithSandboxName(sandboxName).WithData(sandboxSpec)
	result, err := cfg.Client.Sandboxes.ApplySandbox(params, nil)
	if err != nil {
		return err
	}
	resp := result.Payload

	fmt.Fprintf(log, "Created sandbox %q (routing key: %s) in cluster %q.\n\n",
		sandboxName, resp.RoutingKey, cfg.Cluster)

	if cfg.Wait {
		// Wait for the sandbox to be ready
		resp, err = waitForReadyCreate(cfg, log, resp)
		if err != nil {
			writeOutputCreate(cfg, out, resp)
			fmt.Fprintf(log, "\nThe sandbox was created, but it may not be ready yet. To check status, run:\n\n")
			fmt.Fprintf(log, "  signadot sandbox get %v\n\n", sandboxName)
			return err
		}
		writeOutputCreate(cfg, out, resp)
		fmt.Fprintf(log, "\nThe sandbox %q was created and is ready.\n", resp.Name)
		return nil
	}
	return writeOutputCreate(cfg, out, resp)
}

func parseKubernetesWorkload(workload string) (kind, namespace, name string, err error) {
	parts := strings.Split(workload, "/")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("expected format KIND/NAMESPACE/NAME, got %q", workload)
	}
	return parts[0], parts[1], parts[2], nil
}

func generateSandboxName(workloadName string) string {
	// Generate a unique sandbox name based on workload name and timestamp
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("%s-%s", workloadName, timestamp)
}

func writeOutputCreate(cfg *config.SandboxCreate, out io.Writer, resp *models.Sandbox) error {
	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		// Print info on how to access the sandbox
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

func waitForReadyCreate(cfg *config.SandboxCreate, out io.Writer, sb *models.Sandbox) (*models.Sandbox, error) {
	fmt.Fprintf(out, "Waiting (up to --wait-timeout=%v) for sandbox to be ready...\n", cfg.WaitTimeout)

	params := sandboxes.NewGetSandboxParams().WithOrgName(cfg.Org).WithSandboxName(sb.Name)

	// Simple polling implementation (can be enhanced with spinner like in apply.go)
	timeout := time.After(cfg.WaitTimeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return sb, fmt.Errorf("timeout waiting for sandbox to be ready")
		case <-ticker.C:
			result, err := cfg.Client.Sandboxes.GetSandbox(params, nil)
			if err != nil {
				continue // Keep retrying
			}
			sb = result.Payload
			if sb.Status.Ready {
				return sb, nil
			}
		}
	}
}
