package override

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/signadot/cli/internal/builder"
	"github.com/signadot/cli/internal/config"
	sbmgr "github.com/signadot/cli/internal/locald/sandboxmanager"
	"github.com/signadot/cli/internal/poll"
	"github.com/signadot/cli/internal/spinner"
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
  signadot local override --sandbox=my-sandbox --to=localhost:9999 --detach`,
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

	workloadNames, err := getOverrideWorkloadNames(sandbox, cfg.Workloads)
	if err != nil {
		return err
	}

	_, err = sbmgr.ValidateSandboxManager(sandbox.Spec.Cluster)
	if err != nil {
		return err
	}

	sandbox, overrideName, err := createSandboxWithMiddleware(cfg, sandbox, workloadNames)
	if err != nil {
		return err
	}

	sandbox, err = utils.WaitForSandboxReady(cfg.API, out, cfg.Sandbox, cfg.WaitTimeout)
	if err != nil {
		return err
	}

	err = waitForLocalReady(out, cfg)
	if err != nil {
		return err
	}

	if cfg.Detach {
		if len(workloadNames) == 1 {
			fmt.Fprintf(out, "Overriding traffic from sandbox '%s' workload '%s' to %s\n", cfg.Sandbox, workloadNames[0], cfg.To)
		} else {
			fmt.Fprintf(out, "Overriding traffic from sandbox '%s' workloads '%s' to %s\n", cfg.Sandbox, strings.Join(workloadNames, ", "), cfg.To)
		}

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
		if err := deleteSandboxWithMiddleware(cfg, sandbox, overrideName); err != nil {
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

// getOverrideWorkloadNames returns the workload names for the given target workload. If no target workload is provided, all workload names are returned.
// If a target workload is provided, but not found, an error is returned.
func getOverrideWorkloadNames(sandbox *models.Sandbox, targetWorkloads []string) ([]string, error) {
	workloadNames := make([]string, 0)

	if len(targetWorkloads) == 0 {
		for _, fork := range sandbox.Spec.Forks {
			workloadNames = append(workloadNames, fork.Name)
		}
	} else {
		for _, target := range targetWorkloads {
			found := false
			for _, fork := range sandbox.Spec.Forks {
				if fork.Name == target {
					workloadNames = append(workloadNames, fork.Name)
					found = true
					continue
				}
			}

			if !found {
				return nil, fmt.Errorf("workload %s not found in sandbox %s", target, sandbox.Name)
			}
		}
	}

	return workloadNames, nil
}

func createSandboxWithMiddleware(cfg *config.LocalOverrideCreate, baseSandbox *models.Sandbox, workloadNames []string) (*models.Sandbox, string, error) {
	sbBuilder := builder.
		BuildSandbox(cfg.Sandbox, builder.WithData(*baseSandbox)).
		AddOverrideMiddleware(cfg.Port, cfg.To, workloadNames...).
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

func deleteSandboxWithMiddleware(cfg *config.LocalOverrideCreate, sandbox *models.Sandbox, overrideName string) error {
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

func waitForLocalReady(out io.Writer, cfg *config.LocalOverrideCreate) error {
	fmt.Fprintf(out, "Waiting (up to --wait-timeout=%v) for local to be ready...\n", cfg.WaitTimeout)

	spin := spinner.Start(out, "Local status")
	defer spin.Stop()

	retry := poll.
		NewPoll().
		WithTimeout(cfg.WaitTimeout)

	var lastErr error
	err := retry.Until(func() poll.PollingState {
		// Ping to port cfg.To
		conn, err := net.DialTimeout("tcp", cfg.To, 1*time.Second)
		spin.Messagef("%v", err)

		if err != nil {
			lastErr = err
			return poll.KeepPolling
		}

		conn.Close()
		return poll.StopPolling
	})

	if lastErr != nil {
		spin.StopFail()
		return lastErr
	}

	if err != nil {
		spin.StopFail()
		return err
	}

	return nil
}
