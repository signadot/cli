package override

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/signadot/cli/internal/builder"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func New(local *config.Local) *cobra.Command {
	cfg := &config.LocalOverrideCreate{LocalOverride: &config.LocalOverride{Local: local}}

	cmd := &cobra.Command{
		Use:   "override --sandbox=<sandbox> --to=<target> [--detach]",
		Short: "Override traffic routing for sandboxes",
		Long: `Override traffic routing allows you to redirect traffic from a sandbox to a local service.
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

	sandbox, err = createSandboxWithMiddleware(cfg, sandbox)
	if err != nil {
		return err
	}

	// TODO: Implement actual override creation logic
	// This is a skeleton implementation

	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	white := color.New(color.FgHiWhite, color.Bold).SprintFunc()

	printOverrideProgress(out, fmt.Sprintf("Redirecting traffic from %s to %s", cfg.Sandbox, cfg.To))

	if !cfg.Detach {
		fmt.Fprintf(out, "* Redirect changes will be removed once session is terminated\n")
		fmt.Fprintf(out, "Press %s to terminate redirect\n\n", white("Ctrl+C"))
	} else {
		fmt.Fprintf(out, "* Redirect changes will be preserved after session is terminated\n")
	}

	printOverrideProgress(out, "Waiting for sandbox ready")

	// Simulate some activity
	time.Sleep(1 * time.Second)
	fmt.Fprintf(out, "POST    /locations -> %s (local)\n", green("Ok"))
	time.Sleep(500 * time.Millisecond)
	fmt.Fprintf(out, "POST    /locations -> %s (local)\n", green("Ok"))
	time.Sleep(500 * time.Millisecond)
	fmt.Fprintf(out, "GET     /locations/test -> %s (sandbox)\n", yellow("Skipped"))

	if !cfg.Detach {
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
			// TODO: Implement cleanup logic
		case <-ctx.Done():
			// Context was cancelled
		}
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

func createSandboxWithMiddleware(cfg *config.LocalOverrideCreate, baseSandbox *models.Sandbox) (*models.Sandbox, error) {
	sb := builder.BuildSandbox(cfg.Sandbox, builder.WithData(*baseSandbox))
	sb.AddOverrideMiddleware(80, cfg.To, "locations")

	sbData := sb.Build()

	sbParams := sandboxes.
		NewApplySandboxParams().
		WithOrgName(cfg.Org).
		WithSandboxName(cfg.Sandbox).
		WithData(&sbData)

	resp, err := cfg.Client.Sandboxes.ApplySandbox(sbParams, nil)
	if err != nil {
		return nil, err
	}
	return resp.Payload, nil
}
