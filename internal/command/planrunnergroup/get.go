package planrunnergroup

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	planrunnergroups "github.com/signadot/go-sdk/client/plan_runner_groups"
	"github.com/spf13/cobra"
)

func newGet(prg *config.PlanRunnerGroup) *cobra.Command {
	cfg := &config.PlanRunnerGroupGet{PlanRunnerGroup: prg}

	cmd := &cobra.Command{
		Use:   "get NAME",
		Short: "Get plan runner group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return get(cfg, cmd.OutOrStdout(), args[0])
		},
	}

	return cmd
}

func get(cfg *config.PlanRunnerGroupGet, out io.Writer, name string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	params := planrunnergroups.NewGetPlanrunnergroupParams().
		WithOrgName(cfg.Org).
		WithPlanRunnerGroupName(name)
	resp, err := cfg.Client.PlanRunnerGroups.GetPlanrunnergroup(params, nil)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printPlanRunnerGroupDetails(out, resp.Payload)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
