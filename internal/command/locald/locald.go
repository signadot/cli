package locald

import (
	"fmt"
	"os"
	"os/exec"
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
	if err := cfg.InitLocalDaemon(); err != nil {
		return err
	}
	ciConfig := cfg.ConnectInvocationConfig
	log, err := getLogger(ciConfig)
	if err != nil {
		return err
	}

	if cfg.DaemonRun {
		// we should spawn a background process
		cmd := exec.Command(os.Args[0], "locald")
		cmd.Env = append(cmd.Env,
			fmt.Sprintf("HOME=%s", ciConfig.UIDHome),
			fmt.Sprintf("PATH=%s", ciConfig.UIDPath),
			fmt.Sprintf("SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG=%s",
				os.Getenv("SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG")),
		)
		if err := cmd.Start(); err != nil {
			return err
		}
		// and check to see that it succeeded
		return processes.WaitReady(ciConfig.GetPIDfile(), time.Second, cmd.Process, log)
	}

	// write our pidfile
	pidFile := ciConfig.GetPIDfile()
	if err := processes.WritePIDFile(pidFile); err != nil {
		return err
	}
	// make sure pidfile perms are correct
	if err := os.Chown(pidFile, ciConfig.UID, ciConfig.GID); err != nil {
		log.Warn("couldn't change ownership of pidfile", "error", err)
	}
	defer func() {
		os.Remove(pidFile)
	}()

	// run the corresponding manager
	if ciConfig.Unprivileged {
		return locald.RunSandboxManager(cfg, log, args)
	}
	return locald.RunAsRoot(cfg, log, args)
}

func getLogger(ciConfig *config.ConnectInvocationConfig) (*slog.Logger, error) {
	logWriter, _, err := system.GetRollingLogWriter(
		ciConfig.SignadotDir,
		ciConfig.GetLogName(),
		ciConfig.UID,
		ciConfig.GID,
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
