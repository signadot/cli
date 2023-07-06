package locald

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/locald"
	"github.com/signadot/cli/internal/utils/system"
	"github.com/signadot/libconnect/common/processes"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
)

func New(apiConfig *config.API) *cobra.Command {
	cfg := &config.LocalDaemon{}

	cmd := &cobra.Command{
		Use:    "locald",
		Short:  "local controller",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cfg, args)
		},
	}
	cfg.AddFlags(cmd)

	return cmd
}

func run(cfg *config.LocalDaemon, args []string) error {
	if err := cfg.InitLocalDaemon(); err != nil {
		return err
	}

	if cfg.DaemonRun {
		// we should spawn a background process and exit
		cmd := exec.Command(os.Args[0], "locald")
		cmd.Env = append(cmd.Env,
			fmt.Sprintf("HOME=%s", cfg.ConnectInvocationConfig.UIDHome),
			fmt.Sprintf("PATH=%s", cfg.ConnectInvocationConfig.UIDPath),
			fmt.Sprintf("SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG=%s",
				os.Getenv("SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG")),
		)
		return cmd.Start()
	}

	// write our pidfile
	pidFile := cfg.ConnectInvocationConfig.GetPidfile()
	processes.WritePIDFile(pidFile, os.Getpid())
	defer func() {
		os.Remove(pidFile)
	}()

	// setup logging
	logWriter, _, err := system.GetRollingLogWriter(
		cfg.ConnectInvocationConfig.SignadotDir,
		cfg.ConnectInvocationConfig.GetLogName(),
		cfg.ConnectInvocationConfig.UID,
		cfg.ConnectInvocationConfig.GID,
	)
	if err != nil {
		return fmt.Errorf("couldn't open logfile, %w", err)
	}
	logLevel := slog.LevelInfo
	if cfg.ConnectInvocationConfig.Debug {
		logLevel = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(logWriter, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// run the corresponding manager
	if cfg.ConnectInvocationConfig.Unprivileged {
		return locald.RunSandboxManager(cfg, log, args)
	}
	return locald.RunAsRoot(cfg, log, args)
}
