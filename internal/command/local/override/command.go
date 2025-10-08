package override

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
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
	"github.com/signadot/libconnect/common/override"
	"github.com/spf13/cobra"
)

func New(local *config.Local) *cobra.Command {
	cfg := &config.LocalOverrideCreate{LocalOverride: &config.LocalOverride{Local: local}}

	cmd := &cobra.Command{
		Use:   "override --sandbox=<sandbox> --to=<target> --port=<port> [--detach] [--workload=<workload>]",
		Short: "Override traffic routing for sandboxes",
		Long: `Override traffic routing allows you to route traffic from a sandbox to a local service.
This is useful for testing local changes against a sandbox environment.
`,
		Example: `  signadot local override --sandbox=my-sandbox --to=9999 --port=8080
  signadot local override --sandbox=my-sandbox --to=localhost:9999 --port=8080 --detach
  signadot local override list
  signadot local override delete <name> --sandbox=<sandbox>
  signadot local override --sandbox=my-sandbox --to=localhost:9999 --port=8080 --workload=my-workload`,
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

	var (
		logServer   *http.Server
		logListener net.Listener
		logOutChan  chan override.LogEntry
	)
	logPort := int64(0)
	if !cfg.Detach {
		logOutChan = make(chan override.LogEntry)
		logServer, logListener, logPort = createLogServer(logOutChan)

		go func() {
			for logEntry := range logOutChan {
				printFormattedLogEntry(logEntry, cfg.Sandbox, cfg.To)
			}
		}()
	}

	sandbox, overrideName, err := createSandboxWithMiddleware(cfg, sandbox, workloadName, logPort)
	if err != nil {
		return err
	}

	sandbox, err = utils.WaitForSandboxReady(cfg.API, out, cfg.Sandbox, cfg.WaitTimeout)
	if err != nil {
		return err
	}

	if !cfg.Detach {
		startLogServer(logServer, logListener)
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

		// Shutdown log server gracefully
		if logServer != nil {
			logServer.Shutdown(ctx)
		}

		// Close log channel to stop the consumer goroutine
		if logOutChan != nil {
			close(logOutChan)
		}
	case <-ctx.Done():
		// Context was cancelled
		if logServer != nil {
			logServer.Shutdown(ctx)
		}
		// Close log channel to stop the consumer goroutine
		if logOutChan != nil {
			close(logOutChan)
		}
	}

	return nil
}

// createLogServer creates an HTTP server and listener for log consumption
// Returns the server, listener, and the actual port that was assigned
func createLogServer(logOutChan chan override.LogEntry) (*http.Server, net.Listener, int64) {
	mux := http.NewServeMux()

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("error listening on available port: %v", err)
	}

	// Get the actual port that was assigned
	listeningPort := int64(ln.Addr().(*net.TCPAddr).Port)

	mux.HandleFunc("/api/logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read the log body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		var logEntry override.LogEntry
		if err := json.Unmarshal(body, &logEntry); err != nil {
			http.Error(w, "failed to unmarshal body", http.StatusInternalServerError)
			return
		}

		logOutChan <- logEntry

		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Handler: mux,
	}

	return server, ln, listeningPort
}

// startLogServer starts an HTTP server with the provided listener
func startLogServer(server *http.Server, ln net.Listener) {
	go func() {
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("log server error: %v", err)
		}
	}()
}

func printFormattedLogEntry(logEntry override.LogEntry, sandboxName string, localAddress string) {
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

func createSandboxWithMiddleware(cfg *config.LocalOverrideCreate, baseSandbox *models.Sandbox, workloadName string, logHost int64) (*models.Sandbox, string, error) {

	policy, err := builder.NewOverrideArgPolicy(cfg.PolicyDefaultFallThroughStatus, cfg.PolicyUnimplementedResponseCodes)
	if err != nil {
		return nil, "", err
	}

	var log *builder.MiddlewareOverrideArg
	if logHost > 0 {
		log, err = builder.NewOverrideLogArg(logHost)
		if err != nil {
			return nil, "", err
		}
	}

	sbBuilder := builder.
		BuildSandbox(cfg.Sandbox, builder.WithData(*baseSandbox)).
		AddOverrideMiddleware(cfg.Port, cfg.To, []string{workloadName}, policy, log).
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
