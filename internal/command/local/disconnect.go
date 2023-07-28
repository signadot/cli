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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func newDisconnect(localConfig *config.Local) *cobra.Command {
	cfg := &config.LocalDisconnect{Local: localConfig}

	cmd := &cobra.Command{
		Use:   "disconnect",
		Short: "disconnect local machine from cluster",
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
	return runDisconnectWith(cfg, signadotDir)
}

func runDisconnectWith(cfg *config.LocalDisconnect, signadotDir string) error {

	runState := &runState{}
	ticker := time.NewTicker(time.Second / 10)
	defer ticker.Stop()
	wasRunning := false
	for {
		wasRunning = runState.init(signadotDir) || wasRunning
		err := runState.tryKill()
		_ = err
		if runState.isDone() {
			break
		}
		<-ticker.C
	}

	if wasRunning {
		fmt.Printf("disconnected.\n")
		return nil
	}
	return fmt.Errorf("signadot was not connected")
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
	if rs.NotRootPIDFilePresent {
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
	_, err := os.Stat(rmPIDFile)
	rs.RootPIDFilePresent = err == nil || !os.IsNotExist(err)
	_, err = os.Stat(sbmPIDFile)
	rs.NotRootPIDFilePresent = err == nil || !os.IsNotExist(err)
	return rs.RootPIDFilePresent || rs.NotRootPIDFilePresent
}
