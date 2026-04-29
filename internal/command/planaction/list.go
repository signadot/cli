package planaction

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	planactions "github.com/signadot/go-sdk/client/plan_actions"
	"github.com/spf13/cobra"
)

func newList(action *config.PlanAction) *cobra.Command {
	cfg := &config.PlanActionList{PlanAction: action}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List plan actions",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return list(cfg, cmd.OutOrStdout())
		},
	}

	return cmd
}

func list(cfg *config.PlanActionList, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	params := planactions.NewListPlanActionsParams().WithOrgName(cfg.Org)
	resp, err := cfg.Client.PlanActions.ListPlanActions(params, nil)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printActionTable(out, resp.Payload)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
