package planrunnergroup

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	planrunnergroups "github.com/signadot/go-sdk/client/plan_runner_groups"
	"github.com/spf13/cobra"
)

func newList(prg *config.PlanRunnerGroup) *cobra.Command {
	cfg := &config.PlanRunnerGroupList{PlanRunnerGroup: prg}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List plan runner groups",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return list(cfg, cmd.OutOrStdout())
		},
	}

	return cmd
}

func list(cfg *config.PlanRunnerGroupList, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	resp, err := cfg.Client.PlanRunnerGroups.ListPlanrunnergroup(
		planrunnergroups.NewListPlanrunnergroupParams().WithOrgName(cfg.Org), nil)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printPlanRunnerGroupTable(out, resp.Payload)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
