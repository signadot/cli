package local

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/utils/system"
	"github.com/signadot/libconnect/common/processes"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newConnect(localConfig *config.Local) *cobra.Command {
	cfg := &config.LocalConnect{Local: localConfig}
	_ = cfg

	cmd := &cobra.Command{
		Use:   "connect",
		Short: "connect with sandboxes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConnect(cmd, cmd.ErrOrStderr(), cfg, args)
		},
	}
	cfg.AddFlags(cmd)

	return cmd
}

func runConnect(cmd *cobra.Command, log io.Writer, cfg *config.LocalConnect, args []string) error {
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
		Unprivileged:     cfg.Unprivileged,
		SignadotDir:      signadotDir,
		APIPort:          6666,
		LocalNetPort:     6667,
		Cluster:          cfg.Cluster,
		UID:              os.Geteuid(),
		UIDHome:          homeDir,
		UIDPath:          os.Getenv("PATH"),
		API:              cfg.API,
		APIKey:           viper.GetString("api_key"),
		ConnectionConfig: connConfig,
	}

	if cfg.NonInteractive {
		return runNonInteractiveConnect(log, cfg, ciConfig)
	}
	return nil

	// ctx := context.Background()

	// log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
	// 	Level: slog.LevelDebug,
	// }))

	// proc, err := processes.NewRetryProcess(ctx, &processes.RetryProcessConf{
	// 	Log: log,
	// 	GetCmd: func() *exec.Cmd {
	// 		var cmdToRun *exec.Cmd
	// 		if !cfg.Unprivileged {
	// 			cmdToRun = exec.Command(
	// 				"sudo",
	// 				"-S",
	// 				"--preserve-env=SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG",
	// 				os.Args[0],
	// 				"locald",
	// 			)
	// 			cmdToRun.SysProcAttr = &syscall.SysProcAttr{
	// 				Setsid: true,
	// 			}
	// 		} else {
	// 			cmdToRun = exec.Command(os.Args[0], "locald")
	// 			cmdToRun.Env = append(cmdToRun.Env,
	// 				fmt.Sprintf("HOME=%s", runConfig.UIDHome),
	// 				fmt.Sprintf("PATH=%s", runConfig.UIDPath))
	// 		}
	// 		cmdToRun.Env = append(cmdToRun.Env,
	// 			fmt.Sprintf("SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG=%s", string(runConfigBytes)))
	// 		cmdToRun.Stderr = os.Stderr
	// 		cmdToRun.Stdout = os.Stdout
	// 		cmdToRun.Stdin = os.Stdin
	// 		return cmdToRun
	// 	},
	// 	Shutdown: func(process *os.Process, runningCh chan struct{}) error {
	// 		if cfg.Unprivileged {
	// 			return nil
	// 		}
	// 		log.Debug("local connect shutdown")

	// 		// Establish a connection with root manager
	// 		grpcConn, err := grpc.Dial("127.0.0.1:6667", grpc.WithTransportCredentials(insecure.NewCredentials()))
	// 		if err != nil {
	// 			return fmt.Errorf("couldn't connect root api, %v", err)
	// 		}
	// 		defer grpcConn.Close()

	// 		// Send the shutdown order
	// 		rootclient := rootapi.NewRootManagerAPIClient(grpcConn)
	// 		if _, err = rootclient.Shutdown(ctx, &rootapi.ShutdownRequest{}); err != nil {
	// 			return fmt.Errorf("error requesting shutdown in root api, %v", err)
	// 		}

	// 		// Wait until shutdown
	// 		select {
	// 		case <-runningCh:
	// 		case <-time.After(5 * time.Second):
	// 			// Kill the process and wait until it's gone
	// 			if err := process.Kill(); err != nil {
	// 				return fmt.Errorf("failed to kill process: %w ", err)
	// 			}
	// 			<-runningCh
	// 		}
	// 		return nil
	// 	},
	// 	PIDFile: pidFile,
	// })
	// if err != nil {
	// 	return err
	// }

	// // Wait until termination
	// sigs := make(chan os.Signal, 1)
	// signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	// <-sigs

	// return proc.Stop()
}

func runNonInteractiveConnect(log io.Writer, cfg *config.LocalConnect,
	ciConfig *config.ConnectInvocationConfig) error {
	// Check if the corresponding manager is already running
	isRunning, err := processes.IsDeamonRunning(ciConfig.GetPidfile())
	if err != nil {
		return err
	}
	if isRunning {
		fmt.Fprintf(log, "signadot is already connected")
		return nil
	}

	// Run signadot locald
	ciConfigBytes, err := json.Marshal(ciConfig)
	if err != nil {
		// should be impossible
		return err
	}

	var cmd *exec.Cmd
	if !cfg.Unprivileged {
		cmd = exec.Command(
			"sudo",
			"--preserve-env=SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG",
			os.Args[0],
			"locald",
			"--deamon",
		)
	} else {
		cmd = exec.Command(os.Args[0], "locald", "--deamon")
		cmd.Env = append(cmd.Env,
			fmt.Sprintf("HOME=%s", ciConfig.UIDHome),
			fmt.Sprintf("PATH=%s", ciConfig.UIDPath),
		)
	}
	cmd.Env = append(cmd.Env,
		fmt.Sprintf("SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG=%s", string(ciConfigBytes)))
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("couldn't run signadot locald, %w", err)
	}
	return nil
}
