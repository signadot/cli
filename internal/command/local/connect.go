package local

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/signadot/cli/internal/config"
	sbmapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	"github.com/signadot/cli/internal/utils/system"
	"github.com/signadot/libconnect/common/processes"
	connectcfg "github.com/signadot/libconnect/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/exp/slog"
	"sigs.k8s.io/yaml"
)

func newConnect(localConfig *config.Local) *cobra.Command {
	cfg := &config.LocalConnect{Local: localConfig}

	cmd := &cobra.Command{
		Use:   "connect",
		Short: "connect local machine to cluster",
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

	// Resolve the user
	user, err := user.Current()
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
		WithRootManager: !cfg.Unprivileged,
		SignadotDir:     signadotDir,
		APIPort:         6666,
		LocalNetPort:    6667,
		User: &config.ConnectInvocationUser{
			UID:      os.Geteuid(),
			GID:      os.Getegid(),
			UIDHome:  user.HomeDir,
			UIDPath:  os.Getenv("PATH"),
			Username: user.Username,
		},
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

	return runConnectImpl(out, logger, cfg, ciConfig)
}

func runConnectImpl(out io.Writer, log *slog.Logger, localConfig *config.LocalConnect, ciConfig *config.ConnectInvocationConfig) error {
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
	// find the path to the current named program, so when we call
	// it is is independent of PATH.
	binary, err := exec.LookPath(os.Args[0])
	if err != nil {
		return fmt.Errorf("unable to find executable %q: %w", os.Args[0], err)
	}

	var cmd *exec.Cmd
	if ciConfig.WithRootManager {
		if os.Geteuid() != 0 {
			fmt.Fprintf(out, "signadot local connect needs root privileges for:\n\t"+
				"- updating /etc/hosts with cluster service names\n\t"+
				"- configuring networking to direct local traffic to the cluster\n")
		}
		// run the root-manager
		cmd = exec.Command(
			"sudo",
			"--preserve-env=SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG",
			binary,
			"locald",
			"--daemon",
			"--root-manager",
		)
	} else {
		// run the sandbox-manager
		cmd = exec.Command(
			binary, "locald",
			"--daemon",
			"--sandbox-manager",
		)
		cmd.Env = append(cmd.Env,
			fmt.Sprintf("HOME=%s", ciConfig.User.UIDHome),
			fmt.Sprintf("PATH=%s", ciConfig.User.UIDPath),
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
	if !localConfig.Wait {
		fmt.Fprintf(out, "you can check its status with: %s\n", white("signadot local status"))
		return nil
	}
	return waitConnect(localConfig, out)
}

func waitConnect(localConfig *config.LocalConnect, out io.Writer) error {
	var (
		ticker   = time.NewTicker(time.Second / 10)
		deadline = time.After(localConfig.WaitTimeout)
		status   *sbmapi.StatusResponse
		err      error
	)
	defer ticker.Stop()
	for {
		status, err = getStatus()
		if err != nil {
			fmt.Fprintf(out, "error getting status: %s", err.Error())
			goto tick
		}
		if isSuccess(status) {
			break
		}
	tick:
		select {
		case <-ticker.C:
		case <-deadline:
			goto doneWaiting
		}
	}
doneWaiting:

	printLocalStatus(&config.LocalStatus{
		Local: localConfig.Local,
	}, out, status)
	if isSuccess(status) {
		return nil
	}
	return fmt.Errorf("connect failed")
}

// TODO unify this with printLocalStatus
func isSuccess(status *sbmapi.StatusResponse) bool {
	ciConfig, err := sbmapi.ToCIConfig(status.CiConfig)
	if err != nil {
		return false
	}

	// check port forward status
	if ciConfig.ConnectionConfig.Type == connectcfg.PortForwardLinkType {
		err := checkPortforwardStatus(status.Portforward)
		if err != nil {
			return false
		}
	}
	// TODO check proxyAddress config

	// check root manager (if running)
	if ciConfig.WithRootManager {
		// check localnet service
		err := checkLocalNetStatus(status.Localnet)
		if err != nil {
			return false
		}
		// check hosts service
		err = checkHostsStatus(status.Hosts)
		if err != nil {
			return false
		}
	}
	return true
}

func getLogger(ciConfig *config.ConnectInvocationConfig) (*slog.Logger, error) {
	logWriter, _, err := system.GetRollingLogWriter(
		ciConfig.SignadotDir,
		ciConfig.GetLogName(ciConfig.WithRootManager),
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
