package override

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/signadot/cli/internal/config"
	sbmgr "github.com/signadot/cli/internal/locald/sandboxmanager"
	"github.com/signadot/cli/internal/poll"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
	"github.com/signadot/libconnect/common/override"
	"github.com/spf13/cobra"
)

func New(local *config.Local) *cobra.Command {
	cfg := &config.LocalOverrideCreate{
		LocalOverride: &config.LocalOverride{Local: local},
	}

	cmd := &cobra.Command{
		Use:   "override --sandbox=<sandbox> [--workload=<workload>] --port=<port> --to=<target> [--except-status=...] [--detach]",
		Short: "Override sandbox HTTP traffic using a local service",
		Long: `Using the 'override' command, when a request comes into the sandbox it is
delivered to the local service. The response of the local service determines
whether or not the request will subsequently be delivered to its original
destination (i.e. the sandbox workload).

When the request is not subsequently delivered to the original destination,
the response from the local service is the response returned to the client.

When the local service is not running, requests will be delivered to the
original destination after failing to communicate with the local service.

By default, overrides apply when the response from the local service
includes the header 'sd-override: true'.

You can use the '--except-status' flag to specify HTTP response codes that
should not be overridden. When set, all other traffic will be overridden
except for the specified status codes, which will fall through to the
original sandboxed destination.`,
		Example: `  # Override sandbox traffic from workload my-workload, port 8080 to localhost:9999
  signadot local override --sandbox=my-sandbox --workload=my-workload --port=8080 --to=localhost:9999

  # Bypass override when the response returns 404 and 503
  signadot local override --sandbox=my-sandbox --workload=my-workload --port=8080 --to=localhost:9999 --except-status=404,503

  # Keep the override active after the CLI session ends
  signadot local override --sandbox=my-sandbox --workload=my-workload --port=8080 --to=localhost:9999 --detach

  # List all active overrides
  signadot local override list

  # Delete a specific override
  signadot local override delete <name> --sandbox=<sandbox>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOverride(cmd.OutOrStdout(), cmd.ErrOrStderr(), cfg)
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

func runOverride(out, errOut io.Writer, cfg *config.LocalOverrideCreate) error {
	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()

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

	// Get the sandbox and validate the workload
	sb, err := utils.GetSandbox(ctx, cfg.API, cfg.Sandbox)
	if err != nil {
		return err
	}
	if err := validateWorkload(sb, cfg.Workload); err != nil {
		return err
	}

	// Make sure sandbox manager is running against the sandbox cluster
	// (signadot local connect has been executed)
	_, err = sbmgr.ValidateSandboxManager(sb.Spec.Cluster)
	if err != nil {
		return err
	}

	// Create the log server (if needed)
	var (
		logServer   *http.Server
		logListener net.Listener
		logPort     int
	)
	if !cfg.Detach {
		logServer, logListener, logPort = createLogServer(cfg.Sandbox, cfg.To)
	}

	// Apply the override to the sandbox
	printOverrideProgress(out, fmt.Sprintf("Applying override to %s", cfg.Sandbox))
	_, overrideName, undo, err := applyOverrideToSandbox(ctx, cfg, sb, cfg.Workload, logPort)
	if err != nil {
		return err
	}

	// NOTE we should keep the single 'retErr' from here down
	var retErr error
	if !cfg.Detach {
		// call the unedit function on exit
		defer func() {
			retErr = errors.Join(retErr, undo(errOut))
		}()
	}

	// Wait until the sandbox is ready
	sb, retErr = utils.WaitForSandboxReady(ctx, cfg.API, out, cfg.Sandbox, cfg.WaitTimeout)
	if retErr != nil {
		return retErr
	}

	if cfg.Detach {
		fmt.Fprintf(out, "Overriding traffic from sandbox %q, workload %q, port %d to %s\n",
			cfg.Sandbox, cfg.Workload, cfg.Port, cfg.To)

		fmt.Fprintf(out, "Traffic override will persist after this session ends\n")

		yellow := color.New(color.FgHiMagenta).SprintFunc()
		helperMessage := fmt.Sprintf("%s local override delete %s --sandbox=%s", os.Args[0], overrideName, cfg.Sandbox)
		fmt.Fprintf(out, "To remove override, run:\n\t%s\n", yellow(helperMessage))

		retErr = nil
		return retErr
	}

	// Start the log server
	startLogServer(ctx, logServer, logListener)

	// Run the readiness loop (until an error or a signal is received)
	readiness := poll.NewPoll().Readiness(ctx, 5*time.Second, ckMatch(ctx, cfg, sb, overrideName))
	defer readiness.Stop()
	retErr = readyLoop(ctx, readiness, errOut)
	return retErr
}

func printFormattedLogEntry(logEntry *override.LogEntry, sandboxName string, localAddress string) {
	var status string
	var routing string

	switch {
	case logEntry.StatusCode >= 200 && logEntry.StatusCode < 300:
		status = color.New(color.FgGreen).Sprintf("%d", logEntry.StatusCode)
	case logEntry.StatusCode >= 300 && logEntry.StatusCode < 400:
		status = color.New(color.FgYellow).Sprintf("%d", logEntry.StatusCode)
	case logEntry.StatusCode >= 400:
		status = color.New(color.FgRed).Sprintf("%d", logEntry.StatusCode)
	default:
		status = fmt.Sprintf("%d", logEntry.StatusCode)
	}

	if logEntry.Overridden {
		routing = color.New(color.FgCyan).Sprint("(" + localAddress + ")")
	} else {
		routing = color.New(color.FgBlue).Sprint("(" + sandboxName + ")")
	}

	fmt.Printf("%-20s %-7s %s -> %s\n",
		routing,
		logEntry.Method,
		logEntry.Path,
		status,
	)
}

func validateWorkload(sandbox *models.Sandbox, workload string) error {
	for _, virtual := range sandbox.Spec.Virtual {
		if virtual.Name == workload {
			return nil
		}
	}

	for _, fork := range sandbox.Spec.Forks {
		if fork.Name == workload {
			return nil
		}
	}

	for _, local := range sandbox.Spec.Local {
		if local.Name == workload {
			return nil
		}
	}

	return fmt.Errorf("workload %s not found in sandbox %s", workload, sandbox.Name)
}
