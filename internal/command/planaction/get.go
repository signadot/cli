package planaction

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	planactions "github.com/signadot/go-sdk/client/plan_actions"
	"github.com/spf13/cobra"
)

func newGet(action *config.PlanAction) *cobra.Command {
	cfg := &config.PlanActionGet{PlanAction: action}

	cmd := &cobra.Command{
		Use:   "get ACTION_NAME",
		Short: "Get a plan action",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return get(cfg, cmd.OutOrStdout(), args[0])
		},
	}

	return cmd
}

func get(cfg *config.PlanActionGet, out io.Writer, name string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	params := planactions.NewGetPlanActionParams().
		WithOrgName(cfg.Org).
		WithActionName(name)
	resp, err := cfg.Client.PlanActions.GetPlanAction(params, nil)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printActionDetails(out, resp.Payload)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
