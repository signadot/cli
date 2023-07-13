package local

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/signadot/cli/internal/config"
	rmapi "github.com/signadot/cli/internal/locald/api/rootmanager"
	sbmapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	"github.com/signadot/cli/internal/utils/system"
	"github.com/signadot/libconnect/common/processes"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func newDisconnect(localConfig *config.Local) *cobra.Command {
	cfg := &config.LocalDisconnect{Local: localConfig}

	cmd := &cobra.Command{
		Use:   "disconnect",
		Short: "disconnect local development with sandboxes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDisconnect(cfg, args)
		},
	}

	return cmd
}

func runDisconnect(cfg *config.LocalDisconnect, args []string) error {
	if err := cfg.InitLocalConfig(); err != nil {
		return err
	}
	signadotDir, err := system.GetSignadotDir()
	if err != nil {
		return err
	}

	runState := &runState{}
	ticker := time.NewTicker(time.Second / 10)
	defer ticker.Stop()
	wasRunning := false
	for !runState.isDone() {
		wasRunning = runState.init(signadotDir) || wasRunning
		err := runState.tryKill()
		_ = err
		<-ticker.C
	}

	if wasRunning {
		fmt.Printf("disconnected.\n")
		return nil
	}
	return fmt.Errorf("signadot was not connected")
}

// process monitor result for one of rootmanager, sandbox manager
type mgrRunState struct {
	C   <-chan struct{}
	PID int
	E   error
}

// we have a sandbox manager and a root manager to stop, and may be
// run in unprivileged mode, without root manager, so
// for stopping them we keep track of both.
type runState struct {
	Root    mgrRunState
	NotRoot mgrRunState
}

// we are done when there are no more pid files
func (rs *runState) isDone() bool {
	return os.IsNotExist(rs.Root.E) && os.IsNotExist(rs.NotRoot.E)
}

func (rs *runState) tryKill() error {
	if rs.Root.C != nil {
		// Establish a connection with sandbox manager
		grpcConn, err := grpc.Dial("127.0.0.1:6667", grpc.WithTransportCredentials(insecure.NewCredentials()))
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
	if rs.NotRoot.C != nil {
		// Establish a connection with root manager
		grpcConn, err := grpc.Dial("127.0.0.1:6666", grpc.WithTransportCredentials(insecure.NewCredentials()))
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

	rs.Root.C, rs.Root.PID, rs.Root.E = processes.MonitorPID(rmPIDFile, slog.Default())
	rs.NotRoot.C, rs.NotRoot.PID, rs.NotRoot.E = processes.MonitorPID(sbmPIDFile, slog.Default())
	return rs.Root.PID != 0 || rs.NotRoot.PID != 0
}
