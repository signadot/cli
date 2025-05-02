package locald

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"net/http"
	_ "net/http/pprof"

	"log/slog"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/locald"
	"github.com/signadot/cli/internal/utils/system"
	"github.com/signadot/libconnect/common/processes"
	"github.com/spf13/cobra"
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
		// find the path to the current named program, so when we call
		// it is is independent of PATH.
		binary, err := exec.LookPath(os.Args[0])
		if err != nil {
			return err
		}
		args := []string{"locald"}
		env := []string{
			fmt.Sprintf("HOME=%s", ciConfig.User.UIDHome),
			fmt.Sprintf("PATH=%s", ciConfig.User.UIDPath),
			fmt.Sprintf("SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG=%s",
				os.Getenv("SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG")),
		}
		if cfg.PProfAddr != "" {
			args = append(args, "--pprof", cfg.PProfAddr)
		}
		if cfg.RootManager {
			args = append(args, "--root-manager")
		} else {
			args = append(args, "--sandbox-manager")
			env = append(env, ciConfig.Env...)
		}
		cmd := exec.Command(binary, args...)
		cmd.Env = env

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

	if cfg.PProfAddr != "" {
		go http.ListenAndServe(cfg.PProfAddr, nil)
	}

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
