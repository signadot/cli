package local

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/signadot/cli/internal/config"
	sbmgr "github.com/signadot/cli/internal/locald/sandboxmanager"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/utils/system"
	"github.com/signadot/libconnect/common/processes"
	"github.com/spf13/cobra"
)

func newStatus(localConfig *config.Local) *cobra.Command {
	cfg := &config.LocalStatus{Local: localConfig}

	cmd := &cobra.Command{
		Use:   "status",
		Short: "show status of the local machine's connection with cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runStatus(cfg, cmd.OutOrStdout(), args); err != nil {
				return print.Error(cmd.OutOrStdout(), err, cfg.OutputFormat)
			}

			return nil
		},
	}
	cfg.AddFlags(cmd)

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

	// Get the status from sandbox manager
	status, err := sbmgr.GetStatus()
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printLocalStatus(cfg, out, status)
	case config.OutputFormatJSON:
		return printRawStatus(cfg, out, print.RawJSON, status)
	case config.OutputFormatYAML:
		return printRawStatus(cfg, out, print.RawK8SYAML, status)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
