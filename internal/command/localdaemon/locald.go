package localdaemon

import (
	"github.com/signadot/cli/internal/config"
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
	if err := cfg.InitLocalConfig(); err != nil {
		return err
	}
	panic("unimplemented")
	return nil
}
