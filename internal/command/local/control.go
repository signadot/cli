package local

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func newControl(localConfig *config.Local) *cobra.Command {
	cfg := &config.LocalControl{Local: localConfig}
	_ = cfg

	cmd := &cobra.Command{
		Use:    "control",
		Short:  "local controller (unprivileged)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			panic("unimplemented")
		},
	}

	return cmd
}
