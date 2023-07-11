package local

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func newStatus(localConfig *config.Local) *cobra.Command {
	cfg := &config.LocalStatus{Local: localConfig}
	_ = cfg

	cmd := &cobra.Command{
		Use:   "status",
		Short: "displays the status about the local development with sandboxes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(cfg, args)
		},
	}

	return cmd
}

func runStatus(cfg *config.LocalStatus, args []string) error {
	if err := cfg.InitLocalConfig(); err != nil {
		return err
	}
	return nil
}
