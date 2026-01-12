package local

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"log/slog"

	"github.com/Masterminds/semver"
	"github.com/fatih/color"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/devbox"
	sbmapi "github.com/signadot/cli/internal/locald/api/sandboxmanager"
	sbmgr "github.com/signadot/cli/internal/locald/sandboxmanager"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/utils/system"
	clusters "github.com/signadot/go-sdk/client/cluster"
	"github.com/signadot/go-sdk/client/devboxes"
	"github.com/signadot/libconnect/common/processes"
	connectcfg "github.com/signadot/libconnect/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sigs.k8s.io/yaml"
)

func newConnect(localConfig *config.Local) *cobra.Command {
	cfg := &config.LocalConnect{Local: localConfig}

	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect local machine to cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runConnect(cmd, cmd.OutOrStdout(), cfg, args); err != nil {
				return print.Error(cmd.OutOrStdout(), err, cfg.OutputFormat)
			}

			return nil
		},
	}
	cfg.AddFlags(cmd)

	return cmd
}

func runConnect(cmd *cobra.Command, out io.Writer, cfg *config.LocalConnect, args []string) error {
	if err := cfg.InitLocalConfig(); err != nil {
		return err
	}

	if cfg.OutputFormat != config.OutputFormatDefault {
		return fmt.Errorf("output format %s not supported for connect", cfg.OutputFormat)
	}

	// Get the sigandot dir and ensure it exists
	signadotDir, err := system.GetSignadotDir()
	if err != nil {
		return err
	}
	err = system.CreateDirIfNotExist(signadotDir)
	if err != nil {
		return err
	}

	// Check if the corresponding manager is already running this gives fail
	// fast response and is safe to return an error here, but the check is _not_
	// used to assume that we have the lock later on when starting.
	withRootManager := !cfg.Unprivileged
	pidFile := config.GetLocaldPIDfile(signadotDir, withRootManager)
	isRunning, err := processes.IsDaemonRunning(pidFile)
	if err != nil {
		return err
	}
	if isRunning {
		return fmt.Errorf("signadot is already connected")
	}

	// We will pass the connConfig to rootmanager and sandboxmanager
	connConfig, err := cfg.GetConnectionConfig(cfg.Cluster)
	if err != nil {
		return err
	}

	// Get devbox claim and session
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var (
		devboxID string
		claimed  bool
	)
	if cfg.Devbox != "" {
		// If devbox ID is provided, validate it exists
		if err := devbox.ValidateDevboxID(ctx, cfg.API, cfg.Devbox); err != nil {
			return err
		}
		devboxID = cfg.Devbox
	} else {
		// If no devbox ID provided, use the stored ID from file
		var err error
		devboxID, err = devbox.GetID(ctx, cfg.API, true, "")
		if err != nil {
			return err
		}
		claimed = true
	}

	if !claimed {
		// Claim the session for the devbox
		params := devboxes.NewClaimDevboxParams().
			WithContext(ctx).
			WithOrgName(cfg.Org).
			WithDevboxID(devboxID)
		_, err := cfg.Client.Devboxes.ClaimDevbox(params)
		if err != nil {
			return fmt.Errorf("failed to claim devbox session: %w", err)
		}
	}

	devboxSessionID, err := devbox.GetSessionID(ctx, cfg.API, devboxID)
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

	// Compute ConnectInvocationConfig
	ciConfig := &config.ConnectInvocationConfig{
		WithRootManager: withRootManager,
		SignadotDir:     signadotDir,
		APIPort:         6666,
		LocalNetPort:    6667,
		VirtualIPNet:    cfg.LocalConfig.VirtualIPNet,
		User: &config.ConnectInvocationUser{
			UID:      os.Geteuid(),
			GID:      os.Getegid(),
			UIDHome:  user.HomeDir,
			UIDPath:  os.Getenv("PATH"),
			Username: user.Username,
		},
		Env:              os.Environ(),
		ConnectionConfig: connConfig,
		ProxyURL:         cfg.ProxyURL,
		APIURL:           cfg.API.APIURL,
		APIKey:           cfg.GetAPIKey(),
		ConfigFile:       viper.ConfigFileUsed(),
		Debug:            cfg.LocalConfig.Debug,
		ConnectTimeout:   cfg.WaitTimeout.String(),
		DevboxID:         devboxID,
		DevboxSessionID:  devboxSessionID,
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

	return runConnectImpl(out, cmd.ErrOrStderr(), logger, cfg, ciConfig)
}

func runConnectImpl(out, errOut io.Writer, log *slog.Logger, localConfig *config.LocalConnect, ciConfig *config.ConnectInvocationConfig) error {
	// Check version skew
	if err := checkVersionSkew(localConfig, ciConfig); err != nil {
		return err
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
		args := []string{
			"--preserve-env=SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG",
			binary,
			"locald",
			"--daemon",
			"--root-manager",
		}
		if localConfig.GOPSAddrRoot != "" {
			args = append(args, "--gops-root-addr", localConfig.GOPSAddrRoot)
		}
		if localConfig.PProfAddr != "" {
			args = append(args, "--pprof", localConfig.PProfAddr)
		}
		cmd = exec.Command("sudo", args...)
	} else {
		// run the sandbox-manager
		cmd = exec.Command(
			binary, "locald",
			"--daemon",
			"--sandbox-manager",
		)
		if localConfig.GOPSAddrNonRoot != "" {
			cmd.Args = append(cmd.Args, "--gops-non-root-addr", localConfig.GOPSAddrNonRoot)
		}
		cmd.Env = append(cmd.Env, ciConfig.Env...)
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
	if localConfig.Wait == config.ConnectWaitNone {
		fmt.Fprintf(out, "you can check its status with: %s\n", white("signadot local status"))
		return nil
	}
	return waitConnect(localConfig, out, errOut)
}

func waitConnect(localConfig *config.LocalConnect, out, errOut io.Writer) error {
	var (
		ciConfig    *config.ConnectInvocationConfig
		ticker      = time.NewTicker(time.Second / 10)
		deadline    = time.After(localConfig.WaitTimeout)
		status      *sbmapi.StatusResponse
		err         error
		connectErrs []error
		sbsOK       bool
	)
	defer ticker.Stop()
	for {
		status, err = sbmgr.GetStatus()
		if err != nil {
			if localConfig.Debug {
				fmt.Fprintf(errOut, "error getting status: %s\n", err.Error())
			}
			connectErrs = []error{err}
			goto tick
		}
		ciConfig, err = sbmapi.ToCIConfig(status.CiConfig)
		if err != nil {
			connectErrs = []error{err}
			goto tick

		}
		// wait until the local connection has been established
		connectErrs = sbmgr.CheckStatusConnectErrors(status, ciConfig)
		if len(connectErrs) != 0 {
			goto tick
		}
		// wait until all local sandboxes are ready (all tunnels have connected)
		if isRunning, lastError := sbmgr.IsWatcherRunning(status); !isRunning {
			if lastError == sbmgr.SandboxesWatcherUnimplemented {
				// this is an old operator, we are done
				break
			}
			goto tick
		}

		sbsOK = true
		for i := range status.Sandboxes {
			sds := status.Sandboxes[i]
			for j := range sds.LocalWorkloads {
				lw := sds.LocalWorkloads[j]
				if lw.TunnelHealth == nil || !lw.TunnelHealth.Healthy {
					sbsOK = false
					goto tick
				}
			}
		}
		break
	tick:
		select {
		case <-ticker.C:
		case <-deadline:
			goto doneWaiting
		}
	}
doneWaiting:

	if status != nil {

		printLocalStatus(&config.LocalStatus{
			Local: localConfig.Local,
		}, out, status)
	} else {
		fmt.Fprintf(out, "could not get local status.\n")
	}

	if len(connectErrs) == 0 {
		switch localConfig.Wait {
		case config.ConnectWaitConnect:
			if !sbsOK {
				fmt.Fprintf(out, "Successfully connected to cluster but some sandboxes are not ready.\n")
			}
			return nil
		case config.ConnectWaitSandboxes:
			if sbsOK {
				return nil
			}
		default:
			// only other option is ConnectWaitDone, which is checked
			// before calling this func.
			panic("unreachable")
		}
	}
	// either connectErrs is non-empty or we requested waiting for
	// sandboxes which aren't ready.  So it failed.  But the connect
	// background process may still be running so we disconnect.
	// Unfortunately, cobra doesn't let you set exit codes so easily, so a
	// caller would have to parse the error message to determine whether or
	// not disconnect on connect failure succeeded.

	// disconnect step 1: Get the sigandot dir
	signadotDir, err := system.GetSignadotDir()
	if err != nil {
		return fmt.Errorf("unable to disconnect failing connect: %w", err)
	}
	// run with initialised config
	if err := runDisconnectWith(&config.LocalDisconnect{
		Local: localConfig.Local,
	}, signadotDir); err != nil {
		return fmt.Errorf("unable to disconnect failing connect: %w", err)
	}
	return fmt.Errorf("connect failed and is no longer running")
}

func checkVersionSkew(localConfig *config.LocalConnect, ciConfig *config.ConnectInvocationConfig) error {
	// get the cluster from API
	params := clusters.NewGetClusterParams().
		WithOrgName(localConfig.Org).WithClusterName(ciConfig.ConnectionConfig.Cluster)
	clusterInfo, err := localConfig.Client.Cluster.GetCluster(params, nil)
	if err != nil {
		return fmt.Errorf("error reading cluster, %w", err)
	}
	if clusterInfo.Payload.Operator == nil {
		return fmt.Errorf("cluster=%s has never connected with Signadot control-plane",
			ciConfig.ConnectionConfig.Cluster)
	}

	// parse the operator version
	operatorVer, err := semver.NewVersion(strings.Split(clusterInfo.Payload.Operator.Version, " ")[0])
	if err != nil {
		return fmt.Errorf("error parsing cluster operator version, %w", err)
	}
	if operatorVer.Prerelease() != "" {
		// this is a pre-release version, let's treat it as a stable version
		ov, err := operatorVer.SetPrerelease("")
		if err != nil {
			return fmt.Errorf("error removing pre-release from cluster operator version, %w", err)
		}
		operatorVer = &ov
	}

	if ciConfig.ConnectionConfig.Type == connectcfg.ControlPlaneProxyLinkType {
		// ControlPlaneProxy requires operator > 0.15.0
		cppConstraint, err := semver.NewConstraint("> 0.15.0")
		if err != nil {
			return err // this shouldn't happen
		}
		if !cppConstraint.Check(operatorVer) {
			return fmt.Errorf("The connection type ControlPlaneProxy requires operator >= v0.16.0")
		}
	}
	return nil
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
