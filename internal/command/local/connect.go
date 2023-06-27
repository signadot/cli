package local

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
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
	// we will pass the connConfig to rootmanager
	// and sandboxmanager
	connConfig, err := cfg.GetConnectionConfig(cfg.Cluster)
	if err != nil {
		return err
	}
	if connConfig.KubeConfigPath == nil {
		kcp := connConfig.GetKubeConfigPath()
		connConfig.KubeConfigPath = &kcp
	}
	// compute ConnectInvocationConfig
	ciConfig := &config.ConnectInvocationConfig{
		Unpriveleged:     cfg.Unpriveleged,
		APIPort:          6666,
		LocalNetPort:     6667,
		Cluster:          cfg.Cluster,
		UID:              os.Geteuid(),
		API:              cfg.API,
		ConnectionConfig: connConfig,
	}
	ciBytes, err := json.Marshal(ciConfig)
	if err != nil {
		// should be impossible
		return err
	}
	var cmdToRun *exec.Cmd
	if !cfg.Unpriveleged {
		cmdToRun = exec.Command("sudo",
			"--preserve-env=SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG",
			os.Args[0], "locald")
	} else {
		cmdToRun = exec.Command(os.Args[0], "locald")
	}
	cmdToRun.Env = append(cmdToRun.Env, fmt.Sprintf("SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG=%s", string(ciBytes)))
	cmdToRun.Env = append(cmdToRun.Env, os.Environ()...)
	cmdToRun.Stderr = os.Stderr
	cmdToRun.Stdout = os.Stdout
	cmdToRun.Stdin = os.Stdin
	fmt.Printf("command: %v\n", cmdToRun)
	return cmdToRun.Run()
}
