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
		return fmt.Errorf("could not init local daemon config: %w", err)
	}
	ciConfig := cfg.ConnectInvocationConfig
	log, err := getLogger(ciConfig, cfg.RootManager)
	if err != nil {
		return fmt.Errorf("could not get logger: %w", err)
	}
	pidFile := ciConfig.GetPIDfile(cfg.RootManager)

	if cfg.DaemonRun {
		// find the path to the current named program, so when we call
		// it is is independent of PATH.
		binary, err := exec.LookPath(os.Args[0])
		if err != nil {
			return fmt.Errorf("locald --daemon: error finding executable path for %s: %w", os.Args[0], err)
		}
		waitTimeout, err := time.ParseDuration(cfg.ConnectInvocationConfig.ConnectTimeout)
		if err != nil {
			return fmt.Errorf("locald --daemon: invalid wait timeout %q: %w", cfg.ConnectInvocationConfigFile, err)

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
			return fmt.Errorf("locald %s --daemon: error starting command binary=%s args=%v: %w", mgr(cfg), binary, cmd.Args, err)
		}
		// and check to see that it succeeded
		if err := processes.WaitReady(pidFile, waitTimeout, cmd.Process, log); err != nil {
			// we can't return an error here because signadot local connect
			// won't get cleaned up but will continue running and can say that it
			// is working (for example airlock tool slows this down).
			log.Warn(fmt.Sprintf("error checking locald %s --daemon sub-process ready", mgr(cfg)), "binary", binary, "args", cmd.Args)
		}
		return nil
	}

	// write our pidfile
	if err := processes.WritePIDFile(pidFile); err != nil {
		return fmt.Errorf("locald %s error writing pid file: %w", mgr(cfg), err)
	}
	// make sure pidfile perms are correct
	if err := os.Chown(pidFile, ciConfig.User.UID, ciConfig.User.GID); err != nil {
		log.Warn("couldn't change ownership of pidfile", "error", err)
	}

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

func mgr(cfg *config.LocalDaemon) string {
	if cfg.RootManager {
		return "root-manager"
	}
	return "sandbox-manager"
}
