package local

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/signadot/cli/internal/config"
	sbmapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/utils/system"
	"github.com/signadot/libconnect/common/processes"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func newStatus(localConfig *config.Local) *cobra.Command {
	cfg := &config.LocalStatus{Local: localConfig}

	cmd := &cobra.Command{
		Use:   "status",
		Short: "displays the status about the local development with sandboxes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(cfg, cmd.OutOrStdout(), args)
		},
	}

	return cmd
}

func runStatus(cfg *config.LocalStatus, out io.Writer, args []string) error {
	if err := cfg.InitLocalConfig(); err != nil {
		return err
	}
	// Get the sigandot dir
	signadotDir, err := system.GetSignadotDir()
	if err != nil {
		return err
	}

	// Make sure the sandbox manager is running
	pidfile := filepath.Join(signadotDir, config.SandboxManagerPIDFile)
	isRunning, err := processes.IsDaemonRunning(pidfile)
	if err != nil {
		return err
	}
	if !isRunning {
		return fmt.Errorf("signadot is not connected\n")
	}

	// Get a sandbox manager API client
	grpcConn, err := grpc.Dial("127.0.0.1:6666", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("couldn't connect sandbox manager api: %w", err)
	}
	defer grpcConn.Close()

	// Send the shutdown order
	sbManagerClient := sbmapi.NewSandboxManagerAPIClient(grpcConn)
	status, err := sbManagerClient.Status(context.Background(), &sbmapi.StatusRequest{})
	if err != nil {
		return fmt.Errorf("couldn't get status from sandbox manager api: %w", err)
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printLocalStatus(cfg, out, status)
	case config.OutputFormatJSON:
		return printRawStatus(out, print.RawJSON, status)
	case config.OutputFormatYAML:
		return printRawStatus(out, print.RawYAML, status)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
