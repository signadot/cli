package local

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func newRootControl(localConfig *config.Local) *cobra.Command {
	cfg := &config.LocalControlRoot{Local: localConfig}
	_ = cfg

	cmd := &cobra.Command{
		Use:    "rootcontrol",
		Short:  "local controller requiring root access",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			panic("unimplemented")
		},
	}

	return cmd
}
