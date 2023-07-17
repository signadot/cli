package local

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/utils/system"
	"github.com/signadot/libconnect/common/processes"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/exp/slog"
	"sigs.k8s.io/yaml"
)

func newConnect(localConfig *config.Local) *cobra.Command {
	cfg := &config.LocalConnect{Local: localConfig}
	_ = cfg

	cmd := &cobra.Command{
		Use:   "connect",
		Short: "connect with sandboxes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConnect(cmd, cmd.OutOrStdout(), cfg, args)
		},
	}
	cfg.AddFlags(cmd)

	return cmd
}

func runConnect(cmd *cobra.Command, out io.Writer, cfg *config.LocalConnect, args []string) error {
	if err := cfg.InitLocalConfig(); err != nil {
		return err
	}

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

	// Get home dir
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
		WithRootManager:  !cfg.Unprivileged,
		SignadotDir:      signadotDir,
		APIPort:          6666,
		LocalNetPort:     6667,
		UID:              os.Geteuid(),
		GID:              os.Getegid(),
		UIDHome:          homeDir,
		UIDPath:          os.Getenv("PATH"),
		ConnectionConfig: connConfig,
		API:              cfg.API,
		APIKey:           viper.GetString("api_key"),
		Debug:            cfg.LocalConfig.Debug,
	}
	if cfg.DumpCIConfig {
		d, _ := yaml.Marshal(ciConfig)
		err := os.WriteFile(filepath.Join(signadotDir, "ci-config.yaml"), d, 0644)
		if err != nil {
			return err
		}
	}
	logger, err := getLogger(ciConfig)
	if err != nil {
		return err
	}

	return runConnectImpl(out, logger, ciConfig)
}

func runConnectImpl(out io.Writer, log *slog.Logger, ciConfig *config.ConnectInvocationConfig) error {
	// Check if the corresponding manager is already running
	// this gives fail fast response and is safe to return
	// an error here, but the check is _not_ used to assume
	// that we have the lock later on when starting.
	pidFile := ciConfig.GetPIDfile(ciConfig.WithRootManager)
	isRunning, err := processes.IsDaemonRunning(pidFile)
	if err != nil {
		return err
	}
	if isRunning {
		return fmt.Errorf("signadot is already connected\n")
	}

	// Run signadot locald
	ciConfigBytes, err := json.Marshal(ciConfig)
	if err != nil {
		// should be impossible
		return err
	}

	var cmd *exec.Cmd
	if ciConfig.WithRootManager {
		if os.Geteuid() != 0 {
			fmt.Fprintf(out, "signadot local connect needs root privileges for:\n\t"+
				"- updating /etc/hosts with cluster service names\n\t"+
				"- configuring networking to direct cluster traffic to the cluster\n")
		}
		// run the root-manager
		cmd = exec.Command(
			"sudo",
			"--preserve-env=SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG",
			os.Args[0],
			"locald",
			"--daemon",
			"--root-manager",
		)
	} else {
		// run the sandbox-manager
		cmd = exec.Command(
			os.Args[0], "locald",
			"--daemon",
			"--sandbox-manager",
		)
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
		return fmt.Errorf("couldn't run signadot locald: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	white := color.New(color.FgHiWhite, color.Bold).SprintFunc()
	fmt.Fprintf(out, "\nsignadot local connect has been started %s\n", green("✓"))
	fmt.Fprintf(out, "you can check its status with: %s\n", white("signadot local status"))
	return nil
}

func getLogger(ciConfig *config.ConnectInvocationConfig) (*slog.Logger, error) {
	logWriter, _, err := system.GetRollingLogWriter(
		ciConfig.SignadotDir,
		ciConfig.GetLogName(ciConfig.WithRootManager),
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
