package local

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/utils/system"
	"github.com/signadot/libconnect/common/processes"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
)

func newConnect(localConfig *config.Local) *cobra.Command {
	cfg := &config.LocalConnect{Local: localConfig}
	_ = cfg

	cmd := &cobra.Command{
		Use:   "connect",
		Short: "connect with sandboxes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConnect(cmd, cfg, args)
		},
	}
	cfg.AddFlags(cmd)

	return cmd
}

func runConnect(cmd *cobra.Command, cfg *config.LocalConnect, args []string) error {
	if err := cfg.InitLocalConfig(); err != nil {
		return err
	}

	// TODO:
	// - define logging
	// - check if another local connect is already running
	// - non-interactive mode
	// - interactive display

	// we will pass the connConfig to rootmanager and sandboxmanager
	connConfig, err := cfg.GetConnectionConfig(cfg.Cluster)
	if err != nil {
		return err
	}

	// Get the sigandot dir
	signadotDir, err := system.GetSignadotDir()
	if err != nil {
		return err
	}
	err = system.CreateDirIfNotExist(signadotDir)
	if err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Set KubeConfigPath if not defined
	if connConfig.KubeConfigPath == nil {
		kcp := connConfig.GetKubeConfigPath()
		connConfig.KubeConfigPath = &kcp
	}

	// compute ConnectInvocationConfig
	ciConfig := &config.ConnectInvocationConfig{
		Unpriveleged:     cfg.Unpriveleged,
		SignadotDir:      signadotDir,
		APIPort:          6666,
		LocalNetPort:     6667,
		Cluster:          cfg.Cluster,
		UID:              os.Geteuid(),
		UIDHome:          homeDir,
		UIDPath:          os.Getenv("PATH"),
		API:              cfg.API,
		ConnectionConfig: connConfig,
	}
	ciBytes, err := json.Marshal(ciConfig)
	if err != nil {
		// should be impossible
		return err
	}

	// Define the pid file name
	var pidFile string
	if !cfg.Unpriveleged {
		pidFile = filepath.Join(signadotDir, config.RootManagerPIDFile)
	} else {
		pidFile = filepath.Join(signadotDir, config.SandboxManagerPIDFile)
	}

	ctx := context.Background()

	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	proc, err := processes.NewRetryProcess(ctx, &processes.RetryProcessConf{
		Log: log,
		GetCmd: func() *exec.Cmd {
			var cmdToRun *exec.Cmd
			if !cfg.Unpriveleged {
				cmdToRun = exec.Command("sudo", "-S",
					"--preserve-env=SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG",
					os.Args[0], "locald")
				// Prevent signaling the children
				cmdToRun.SysProcAttr = &syscall.SysProcAttr{
					Setsid: true,
				}
			} else {
				cmdToRun = exec.Command(os.Args[0], "locald")
				cmdToRun.Env = append(cmdToRun.Env,
					fmt.Sprintf("HOME=%s", ciConfig.UIDHome),
					fmt.Sprintf("PATH=%s", ciConfig.UIDPath))
			}
			cmdToRun.Env = append(cmdToRun.Env,
				fmt.Sprintf("SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG=%s", string(ciBytes)))
			cmdToRun.Stderr = os.Stderr
			cmdToRun.Stdout = os.Stdout
			cmdToRun.Stdin = os.Stdin
			return cmdToRun
		},
		PIDFile: pidFile,
	})
	if err != nil {
		return err
	}

	// Wait until termination
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	if cfg.Unpriveleged {
		return proc.Stop()
	}
	// else we don't have permissions, at least on Mac
	// TODO ping shutdown endpoint
	return nil
}
