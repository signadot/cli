package local

import (
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
			panic("unimplemented")
		},
	}

	return cmd
}
