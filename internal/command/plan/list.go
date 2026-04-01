package plan

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	sdkplans "github.com/signadot/go-sdk/client/plans"
	"github.com/spf13/cobra"
)

func newList(plan *config.Plan) *cobra.Command {
	cfg := &config.PlanList{Plan: plan}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List plans",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listPlans(cfg, cmd.OutOrStdout())
		},
	}

	return cmd
}

func listPlans(cfg *config.PlanList, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	resp, err := cfg.Client.Plans.ListPlans(
		sdkplans.NewListPlansParams().WithOrgName(cfg.Org), nil)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printPlanTable(out, resp.Payload)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
