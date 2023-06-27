package locald

import (
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/locald"
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
	if cfg.ConnectInvocationConfig.Unpriveleged {
		return locald.RunSandboxManager(cfg, args)
	}
	return locald.RunAsRoot(cfg, args)
}
