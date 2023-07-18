package locald

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

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
	signal.Ignore(syscall.SIGHUP)

	if err := cfg.InitLocalDaemon(); err != nil {
		return err
	}
	ciConfig := cfg.ConnectInvocationConfig
	log, err := getLogger(ciConfig, cfg.RootManager)
	if err != nil {
		return err
	}
	pidFile := ciConfig.GetPIDfile(cfg.RootManager)

	if cfg.DaemonRun {
		// we should spawn a background process
		var cmd *exec.Cmd
		if cfg.RootManager {
			cmd = exec.Command(os.Args[0], "locald", "--root-manager")
		} else {
			cmd = exec.Command(os.Args[0], "locald", "--sandbox-manager")
		}
		cmd.Env = append(cmd.Env,
			fmt.Sprintf("HOME=%s", ciConfig.User.UIDHome),
			fmt.Sprintf("PATH=%s", ciConfig.User.UIDPath),
			fmt.Sprintf("SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG=%s",
				os.Getenv("SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG")),
		)
		if err := cmd.Start(); err != nil {
			return err
		}
		// and check to see that it succeeded
		return processes.WaitReady(pidFile, time.Second, cmd.Process, log)
	}

	// write our pidfile
	if err := processes.WritePIDFile(pidFile); err != nil {
		return err
	}
	// make sure pidfile perms are correct
	if err := os.Chown(pidFile, ciConfig.User.UID, ciConfig.User.GID); err != nil {
		log.Warn("couldn't change ownership of pidfile", "error", err)
	}
	defer func() {
		os.Remove(pidFile)
	}()

	// run the corresponding manager
	if cfg.RootManager {
		return locald.RunRootManager(cfg, log, args)
	}
	return locald.RunSandboxManager(cfg, log, args)
}

func getLogger(ciConfig *config.ConnectInvocationConfig, isRootManager bool) (*slog.Logger, error) {
	logWriter, _, err := system.GetRollingLogWriter(
		ciConfig.SignadotDir,
		ciConfig.GetLogName(isRootManager),
		ciConfig.User.UID,
		ciConfig.User.GID,
	)
	if err != nil {
		return nil, fmt.Errorf("couldn't open logfile, %w", err)
	}
	logLevel := slog.LevelInfo
	if ciConfig.Debug {
		logLevel = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(logWriter, &slog.HandlerOptions{
		Level: logLevel,
	}))
	return log, nil
}
