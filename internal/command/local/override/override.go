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
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/signadot/cli/internal/config"
	sbmgr "github.com/signadot/cli/internal/locald/sandboxmanager"
	"github.com/signadot/cli/internal/poll"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
	"github.com/signadot/libconnect/common/override"
)

func runOverride(rootCtx context.Context, out, errOut io.Writer,
	cfg *config.LocalOverrideCreate) error {
	ctx, cancel := signal.NotifyContext(rootCtx,
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
			retErr = errors.Join(retErr, undo(rootCtx, errOut))
		}()
	}

	// Wait until the sandbox is ready
	sb, retErr = utils.WaitForSandboxReady(ctx, cfg.API, out, cfg.Sandbox, cfg.WaitTimeout)
	if retErr != nil {
		return retErr
	}

	if cfg.Detach {
		fmt.Fprintf(out, "All HTTP/gRPC requests intended for sandbox %s, workload %s, port %d will be sent to your local service at %s.\n\n",
			cfg.Sandbox, cfg.Workload, cfg.Port, cfg.To)

		fmt.Fprintf(out, "Traffic override will persist after this session ends\n")

		yellow := color.New(color.FgHiMagenta).SprintFunc()
		helperMessage := fmt.Sprintf("%s local override delete %s --sandbox=%s", os.Args[0], overrideName, cfg.Sandbox)
		fmt.Fprintf(out, "To remove override, run:\n\t%s\n", yellow(helperMessage))

		retErr = nil
		return retErr
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	fmt.Fprintf(out, "%s Local destination %s will override sandbox responses as follows:\n\nAll HTTP/gRPC requests intended for sandbox %s, workload %s, port %d will be sent to your local service at %s.\n\n",
		green("✓"), cfg.To, bold(cfg.Sandbox), bold(cfg.Workload), cfg.Port, bold(cfg.To))

	if len(cfg.ExcludedStatusCodes) > 0 {
		codes := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(cfg.ExcludedStatusCodes)), ","), "[]")
		fmt.Fprintf(out, "* If your local service (%s) responds with status code(s) %s:\n", bold(cfg.To), codes)
		fmt.Fprintf(out, "    -> Request is forwarded to the sandbox (%s).\n", bold(cfg.Sandbox))
		fmt.Fprintf(out, "* Otherwise:\n")
		fmt.Fprintf(out, "    -> Response from your local service (%s) is returned to the client.\n", bold(cfg.To))
	} else {
		fmt.Fprintf(out, "* If your local service (%s) responds with header `sd-override: true`:\n", bold(cfg.To))
		fmt.Fprintf(out, "    -> Response from your local service (%s) is returned to the client.\n", bold(cfg.To))
		fmt.Fprintf(out, "* Otherwise:\n")
		fmt.Fprintf(out, "    -> Request is forwarded to the sandbox (%s).\n", bold(cfg.Sandbox))
	}
	fmt.Fprintf(out, "\n")

	// Inform the user that traffic logs will be printed
	fmt.Fprintln(out, "Printing traffic logs below (press Ctrl+C to stop):")
	fmt.Fprintf(out, "\n")
	// Print the header
	printLogHeader()
	// Start the log server
	startLogServer(ctx, logServer, logListener)

	// Run the readiness loop (until an error or a signal is received)
	readiness := poll.NewPoll().Readiness(ctx, 5*time.Second, ckMatch(ctx, cfg, sb, overrideName))
	defer readiness.Stop()
	retErr = readyLoop(ctx, readiness, errOut)
	return retErr
}

func printLogHeader() {
	bold := color.New(color.Bold).SprintFunc()
	// Pad text first, then apply bold formatting
	servedBy := fmt.Sprintf("%-32s", "SERVED BY")
	method := fmt.Sprintf("%-7s", "METHOD")
	path := "PATH"
	status := "STATUS"
	// Strings are already padded, use %s to avoid ANSI code width issues
	fmt.Printf("%s %s %s -> %s\n",
		bold(servedBy),
		bold(method),
		bold(path),
		bold(status),
	)
}

func printFormattedLogEntry(logEntry *override.LogEntry, sandboxName string, localAddress string) {
	var status string
	var routing string
	var statusStr string
	var routingStr string
	var methodStr string
	var pathStr string

	// Get the plain text first for proper width calculation
	if logEntry.Overridden {
		routingStr = localAddress
	} else {
		routingStr = sandboxName
	}
	routingStr = fmt.Sprintf("%-32s", routingStr)

	statusStr = fmt.Sprintf("%d", logEntry.StatusCode)

	// Display method as-is
	methodStr = logEntry.Method
	methodStr = fmt.Sprintf("%-7s", methodStr)

	// Display path as-is
	pathStr = logEntry.Path

	// Apply colors after padding
	switch {
	case logEntry.StatusCode >= 200 && logEntry.StatusCode < 300:
		status = color.New(color.FgGreen).Sprint(statusStr)
	case logEntry.StatusCode >= 300 && logEntry.StatusCode < 400:
		status = color.New(color.FgYellow).Sprint(statusStr)
	case logEntry.StatusCode >= 400:
		status = color.New(color.FgRed).Sprint(statusStr)
	default:
		status = statusStr
	}

	if logEntry.Overridden {
		routing = color.New(color.FgCyan).Sprint(routingStr)
	} else {
		routing = color.New(color.FgBlue).Sprint(routingStr)
	}

	// Strings are already padded, use %s to avoid ANSI code width issues
	fmt.Printf("%s %s %s -> %s\n",
		routing,
		methodStr,
		pathStr,
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
