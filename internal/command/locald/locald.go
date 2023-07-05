package locald

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/locald"
	"github.com/signadot/libconnect/common/processes"
	"github.com/spf13/cobra"
)

func New(apiConfig *config.API) *cobra.Command {
	cfg := &config.LocalDaemon{Local: &config.Local{API: apiConfig}}

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

	if cfg.DaemonRun {
		// we should spawn a background process and exit
		cmd := exec.Command(os.Args[0], "locald")
		cmd.Env = append(cmd.Env,
			fmt.Sprintf("HOME=%s", cfg.ConnectInvocationConfig.UIDHome),
			fmt.Sprintf("PATH=%s", cfg.ConnectInvocationConfig.UIDPath),
			fmt.Sprintf("SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG=%s",
				os.Getenv("SIGNADOT_LOCAL_CONNECT_INVOCATION_CONFIG")),
		)
		// TODO:
		f, _ := os.OpenFile("/tmp/root-manager.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		cmd.Stderr = f
		cmd.Stdout = f
		return cmd.Start()
	}

	// write our pidfile
	pidFile := cfg.ConnectInvocationConfig.GetPidfile()
	processes.WritePIDFile(pidFile, os.Getpid())
	defer func() {
		os.Remove(pidFile)
	}()

	// run the corresponding manager
	if cfg.ConnectInvocationConfig.Unprivileged {
		return locald.RunSandboxManager(cfg, args)
	}
	return locald.RunAsRoot(cfg, args)
}
