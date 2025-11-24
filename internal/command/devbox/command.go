package devbox

import (
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.Devbox{API: api}

	cmd := &cobra.Command{
		Use:   "devbox",
		Short: "Inspect and manipulate devboxes",
	}

	cmd.AddCommand(
		newList(cfg),
		newRegister(cfg),
		newDelete(cfg),
	)

	return cmd
}
