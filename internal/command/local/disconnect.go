package local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/signadot/cli/internal/config"
	rmapi "github.com/signadot/cli/internal/locald/api/rootmanager"
	sbmapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	sbmgr "github.com/signadot/cli/internal/locald/sandboxmanager"
	"github.com/signadot/cli/internal/utils/system"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/libconnect/common/processes"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func newDisconnect(localConfig *config.Local) *cobra.Command {
	cfg := &config.LocalDisconnect{Local: localConfig}

	cmd := &cobra.Command{
		Use:   "disconnect",
		Short: "Disconnect local machine from cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDisconnect(cfg, args)
		},
	}
	cfg.AddFlags(cmd)

	return cmd
}

func runDisconnect(cfg *config.LocalDisconnect, args []string) error {
	if err := cfg.InitLocalConfig(); err != nil {
		return err
	}
	if cfg.OutputFormat != config.OutputFormatDefault {
		return fmt.Errorf("output format %s not supported for disconnect", cfg.OutputFormat)
	}
	signadotDir, err := system.GetSignadotDir()
	if err != nil {
		return err
	}
	return runDisconnectWith(cfg, signadotDir)
}

func runDisconnectWith(cfg *config.LocalDisconnect, signadotDir string) error {
	if cfg.CleanLocalSandboxes {
		if err := cleanLocalSandboxes(cfg); err != nil {
			return fmt.Errorf("couldn't clean-up local sandboxes, %w", err)
		}
	}

	// perform the disconnect
	runState := &runState{}
	ticker := time.NewTicker(time.Second / 10)
	defer ticker.Stop()
	wasRunning := false
	// There is deliberately no deadline: giving up part way could leave the
	// machine's networking (hosts, routes, pf, resolver) in a bad state. But
	// report what we're waiting on, as the wait may not end on its own (e.g. a
	// stale pidfile whose PID was reused after a hard power-off).
	start := time.Now()
	var lastReport time.Time
	for {
		wasRunning = runState.init(signadotDir) || wasRunning
		err := runState.tryKill()
		if runState.isDone() {
			break
		}
		if time.Since(start) >= disconnectReportAfter && time.Since(lastReport) >= disconnectReportEvery {
			lastReport = time.Now()
			reportDisconnectWait(os.Stderr, signadotDir, runState, time.Since(start), err)
		}
		<-ticker.C
	}
	if wasRunning {
		fmt.Printf("disconnected.\n")
		return nil
	}
	return fmt.Errorf("signadot was not connected")
}

func cleanLocalSandboxes(cfg *config.LocalDisconnect) error {
	// get status from sandboxmanager
	status, err := sbmgr.GetStatus()
	if err != nil {
		if errors.Is(err, sbmgr.ErrSandboxManagerUnavailable) {
			// local is already disconnected (or at least sandboxmanager is not
			// running) there's nothing we can do here, just return
			return nil
		}
		return err
	}

	// init SaaS API
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	for _, sb := range status.Sandboxes {
		// delete the sandbox.
		params := sandboxes.NewDeleteSandboxParams().
			WithOrgName(cfg.Org).
			WithSandboxName(sb.Name)
		_, err := cfg.Client.Sandboxes.DeleteSandbox(params, nil)
		if err != nil {
			return err
		}

		fmt.Printf("Deleted sandbox %q.\n", sb.Name)
	}
	return nil
}

const (
	disconnectReportAfter = 5 * time.Second
	disconnectReportEvery = 30 * time.Second
)

func reportDisconnectWait(w io.Writer, signadotDir string, rs *runState, waited time.Duration, lastErr error) {
	fmt.Fprintf(w, "Still waiting for signadot local daemons to shut down (%s)", waited.Round(time.Second))
	if lastErr != nil {
		fmt.Fprintf(w, ": %v", lastErr)
	}
	fmt.Fprintln(w)
	for _, pf := range []struct {
		present bool
		name    string
	}{
		{rs.RootPIDFilePresent, config.RootManagerPIDFile},
		{rs.NotRootPIDFilePresent, config.SandboxManagerPIDFile},
	} {
		if !pf.present {
			continue
		}
		p := filepath.Join(signadotDir, pf.name)
		pid, err := processes.ReadPIDFile(p)
		if err != nil {
			fmt.Fprintf(w, "  %s: %v\n", p, err)
			continue
		}
		fmt.Fprintf(w, "  %s: pid %d\n", p, pid)
	}
	fmt.Fprintln(w, "If no signadot process has that pid, the pidfile is stale (e.g. after a hard power-off).")
}

// we have a sandbox manager and a root manager to stop, and may be
// run in unprivileged mode, without root manager, so
// for stopping them we keep track of both.
type runState struct {
	RootPIDFilePresent    bool
	NotRootPIDFilePresent bool
}

// we are done when there are no more pid files
func (rs *runState) isDone() bool {
	return !rs.RootPIDFilePresent && !rs.NotRootPIDFilePresent
}

func (rs *runState) tryKill() error {
	if rs.RootPIDFilePresent {
		// Establish a connection with sandbox manager
		grpcConn, err := grpc.NewClient("127.0.0.1:6667", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("couldn't connect sandbox manager api, %w", err)
		}
		defer grpcConn.Close()

		// Send the shutdown order
		rootManagerClient := rmapi.NewRootManagerAPIClient(grpcConn)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err = rootManagerClient.Shutdown(ctx, &rmapi.ShutdownRequest{}); err != nil {
			return fmt.Errorf("error requesting shutdown in root manager api: %w", err)
		}
		return nil
	}
	if rs.NotRootPIDFilePresent {
		// Establish a connection with root manager
		grpcConn, err := grpc.NewClient("127.0.0.1:6666", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("couldn't connect sandbox manager api, %w", err)
		}
		defer grpcConn.Close()

		// Send the shutdown order
		sbManagerClient := sbmapi.NewSandboxManagerAPIClient(grpcConn)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := sbManagerClient.Shutdown(ctx, &sbmapi.ShutdownRequest{}); err != nil {
			return fmt.Errorf("error requesting shutdown in sandbox manager api: %w", err)
		}
	}
	return nil
}

func (rs *runState) init(signadotDir string) bool {
	rmPIDFile := filepath.Join(signadotDir, config.RootManagerPIDFile)
	sbmPIDFile := filepath.Join(signadotDir, config.SandboxManagerPIDFile)
	processes.CleanPIDFile(rmPIDFile)
	processes.CleanPIDFile(sbmPIDFile)
	_, err := os.Stat(rmPIDFile)
	rs.RootPIDFilePresent = err == nil || !os.IsNotExist(err)
	_, err = os.Stat(sbmPIDFile)
	rs.NotRootPIDFilePresent = err == nil || !os.IsNotExist(err)
	return rs.RootPIDFilePresent || rs.NotRootPIDFilePresent
}
