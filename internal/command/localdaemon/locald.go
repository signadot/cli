package localdaemon

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func newLocalDaemon(apiConfig *config.API) *cobra.Command {
	cfg := &config.LocalDaemon{API: apiConfig}

	cmd := &cobra.Command{
		Use:    "locald",
		Short:  "local controller",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			panic("unimplemented")
		},
	}
	cfg.AddFlags(cmd)

	return cmd
}
