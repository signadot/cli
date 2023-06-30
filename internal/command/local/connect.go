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
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/local"
	rootapi "github.com/signadot/cli/internal/locald/api/rootmanager"
	"github.com/signadot/cli/internal/utils/system"
	"github.com/signadot/libconnect/common/processes"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	for i := 0; i < 10; i++ {
		err := local.Lock(signadotDir)
		if err == nil {
			defer func() { local.Unlock(signadotDir) }()
			break
		}
		if cfg.Clobber {
			local.Unlock(signadotDir)
			continue
		}
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
		APIKey:           viper.GetString("api_key"),
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
				cmdToRun = exec.Command(
					"sudo",
					"-S",
					"--preserve-env=SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG",
					os.Args[0],
					"locald",
				)
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
		Shutdown: func(process *os.Process, runningCh chan struct{}) error {
			if cfg.Unpriveleged {
				return nil
			}
			log.Debug("local connect shutdown")

			// Establish a connection with root manager
			grpcConn, err := grpc.Dial("127.0.0.1:6667", grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return fmt.Errorf("couldn't connect root api, %v", err)
			}
			defer grpcConn.Close()

			// Send the shutdown order
			rootclient := rootapi.NewRootManagerAPIClient(grpcConn)
			if _, err = rootclient.Shutdown(ctx, &rootapi.ShutdownRequest{}); err != nil {
				return fmt.Errorf("error requesting shutdown in root api, %v", err)
			}

			// Wait until shutdown
			select {
			case <-runningCh:
			case <-time.After(5 * time.Second):
				// Kill the process and wait until it's gone
				if err := process.Kill(); err != nil {
					return fmt.Errorf("failed to kill process: %w ", err)
				}
				<-runningCh
			}
			return nil
		},
		PIDFile: pidFile,
	})
	if err != nil {
		return err
	}

	// Wait until termination
	<-sigs

	return proc.Stop()
}
