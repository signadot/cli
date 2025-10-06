package override

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/fatih/color"
	"github.com/signadot/cli/internal/builder"
	"github.com/signadot/cli/internal/config"
	sbmgr "github.com/signadot/cli/internal/locald/sandboxmanager"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func New(local *config.Local) *cobra.Command {
	cfg := &config.LocalOverrideCreate{LocalOverride: &config.LocalOverride{Local: local}}

	cmd := &cobra.Command{
		Use:   "override --sandbox=<sandbox> --to=<target> [--detach]",
		Short: "Override traffic routing for sandboxes",
		Long: `Override traffic routing allows you to route traffic from a sandbox to a local service.
This is useful for testing local changes against a sandbox environment.

Examples:
  signadot local override --sandbox=my-sandbox --to=localhost:9999
  signadot local override --sandbox=my-sandbox --to=localhost:9999 --detach
  signadot local override list
  signadot local override delete <name> --sandbox=<sandbox>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOverride(cmd.OutOrStdout(), cfg)
		},
	}

	// Add flags for the main override command
	cfg.AddFlags(cmd)

	// Subcommands
	cmd.AddCommand(
		newDelete(cfg.LocalOverride),
		newList(cfg.LocalOverride),
	)

	return cmd
}

func runOverride(out io.Writer, cfg *config.LocalOverrideCreate) error {
	yellow := color.New(color.FgHiMagenta).SprintFunc()

	if err := cfg.InitLocalConfig(); err != nil {
		return err
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	// Initialize API client
	if err := cfg.API.InitAPIConfig(); err != nil {
		return err
	}

	sandbox, err := getSandbox(cfg)
	if err != nil {
		return err
	}

	workloadName, err := getOverrideWorkloadName(sandbox, cfg.Workload)
	if err != nil {
		return err
	}

	_, err = sbmgr.ValidateSandboxManager(sandbox.Spec.Cluster)
	if err != nil {
		return err
	}

	sandbox, overrideName, err := createSandboxWithMiddleware(cfg, sandbox, workloadName)
	if err != nil {
		return err
	}

	sandbox, err = utils.WaitForSandboxReady(cfg.API, out, cfg.Sandbox, cfg.WaitTimeout)
	if err != nil {
		return err
	}

	if cfg.Detach {
		fmt.Fprintf(out, "Overriding traffic from sandbox '%s' workload '%s' to %s\n", cfg.Sandbox, workloadName, cfg.To)

		fmt.Fprintf(out, "Traffic override will persist after this session ends\n")

		helperMessage := fmt.Sprintf("%s local override delete %s --sandbox=%s", os.Args[0], overrideName, cfg.Sandbox)
		fmt.Fprintf(out, "To remove override, run:\n\t%s\n", yellow(helperMessage))

		return nil
	}

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channel to listen for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal or context cancellation
	select {
	case <-sigChan:
		fmt.Fprintf(out, "\nSession terminated\n")
		printOverrideProgress(out, fmt.Sprintf("Removing redirect in %s", cfg.Sandbox))
		if err := deleteMiddlewareFromSandbox(cfg, sandbox, overrideName); err != nil {
			return err
		}
	case <-ctx.Done():
		// Context was cancelled
	}

	return nil
}

func getSandbox(cfg *config.LocalOverrideCreate) (*models.Sandbox, error) {
	sandboxParams := sandboxes.NewGetSandboxParams().WithOrgName(cfg.Org).WithSandboxName(cfg.Sandbox)

	resp, err := cfg.Client.Sandboxes.
		GetSandbox(sandboxParams, nil)
	if err != nil {
		return nil, err
	}
	return resp.Payload, nil
}

// getOverrideWorkloadName returns the workload name for the given target workload. If no target workload is provided, the first available workload name is returned.
// If a target workload is provided, but not found, an error is returned.
func getOverrideWorkloadName(sandbox *models.Sandbox, targetWorkload string) (string, error) {
	if targetWorkload == "" {
		workloadName, err := getFirstAvailableWorkloadName(sandbox)
		if err != nil {
			return "", err
		}

		return workloadName, nil

	}

	workloadName, err := getWorkloadByName(sandbox, targetWorkload)
	if err != nil {
		return "", err
	}

	return workloadName, nil
}

// getWorkloadByName returns the workload name for the given name
func getWorkloadByName(sandbox *models.Sandbox, name string) (string, error) {

	for _, virtual := range sandbox.Spec.Virtual {
		if virtual.Name == name {
			return virtual.Name, nil
		}
	}

	for _, fork := range sandbox.Spec.Forks {
		if fork.Name == name {
			return fork.Name, nil
		}
	}

	for _, local := range sandbox.Spec.Local {
		if local.Name == name {
			return local.Name, nil
		}
	}

	return "", fmt.Errorf("workload %s not found in sandbox %s", name, sandbox.Name)
}

// getFirstAvailableWorkloadName returns the first available workload name for the given sandbox
// The order is virtual, forks and local
func getFirstAvailableWorkloadName(sandbox *models.Sandbox) (string, error) {
	if len(sandbox.Spec.Virtual) > 0 {
		return sandbox.Spec.Virtual[0].Name, nil
	}

	if len(sandbox.Spec.Forks) > 0 {
		return sandbox.Spec.Forks[0].Name, nil
	}

	if len(sandbox.Spec.Local) > 0 {
		return sandbox.Spec.Local[0].Name, nil
	}

	return "", fmt.Errorf("no available workload found in sandbox %s", sandbox.Name)
}

func createSandboxWithMiddleware(cfg *config.LocalOverrideCreate, baseSandbox *models.Sandbox, workloadName string) (*models.Sandbox, string, error) {
	sbBuilder := builder.
		BuildSandbox(cfg.Sandbox, builder.WithData(*baseSandbox)).
		AddOverrideMiddleware(cfg.Port, cfg.To, workloadName).
		SetMachineID()

	sb, err := sbBuilder.Build()
	if err != nil {
		return nil, "", err
	}

	sbParams := sandboxes.
		NewApplySandboxParams().
		WithOrgName(cfg.Org).
		WithSandboxName(cfg.Sandbox).
		WithData(&sb)

	resp, err := cfg.Client.Sandboxes.ApplySandbox(sbParams, nil)
	if err != nil {
		return nil, "", err
	}

	overrideName := sbBuilder.GetLastAddedOverrideName()

	return resp.Payload, *overrideName, nil
}

func deleteMiddlewareFromSandbox(cfg *config.LocalOverrideCreate, sandbox *models.Sandbox, overrideName string) error {
	sbBuilder := builder.
		BuildSandbox(cfg.Sandbox, builder.WithData(*sandbox)).
		SetMachineID().
		DeleteOverrideMiddleware(overrideName)

	sb, err := sbBuilder.Build()
	if err != nil {
		return err
	}

	if err := cfg.API.RefreshAPIConfig(); err != nil {
		return err
	}

	sbParams := sandboxes.
		NewApplySandboxParams().
		WithOrgName(cfg.Org).
		WithSandboxName(cfg.Sandbox).
		WithData(&sb)

	_, err = cfg.Client.Sandboxes.ApplySandbox(sbParams, nil)
	return err
}
