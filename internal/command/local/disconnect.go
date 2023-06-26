package local

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func newDisconnect(localConfig *config.Local) *cobra.Command {
	cfg := &config.LocalDisconnect{Local: localConfig}
	_ = cfg

	cmd := &cobra.Command{
		Use:   "disconnect",
		Short: "disconnect local development with sandboxes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDisconnect(cfg, args)
		},
	}

	return cmd
}

func runDisconnect(cfg *config.LocalDisconnect, args []string) error {
	if err := cfg.InitLocalConfig(); err != nil {
		return err
	}
	panic("unimplemented")
}
