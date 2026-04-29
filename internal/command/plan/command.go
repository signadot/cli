package plan

import (
	"github.com/signadot/cli/internal/command/planaction"
	"github.com/signadot/cli/internal/command/planexec"
	"github.com/signadot/cli/internal/command/plantag"
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(api *config.API) *cobra.Command {
	cfg := &config.Plan{API: api}

	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Manage plans (compiled prompts that define runnable workflows)",
	}

	// Subcommands
	cmd.AddCommand(
		newCompile(cfg),
		newCreate(cfg),
		newGet(cfg),
		newDelete(cfg),
		newRecompile(cfg),
		plantag.New(cfg),
		planexec.New(cfg),
		planaction.New(cfg),
		newRun(cfg),
	)

	return cmd
}
