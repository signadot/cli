package planexec

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	planexecs "github.com/signadot/go-sdk/client/plan_executions"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newList(exec *config.PlanExecution) *cobra.Command {
	cfg := &config.PlanExecList{PlanExecution: exec}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List plan executions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listExecs(cfg, cmd.OutOrStdout())
		},
	}

	cfg.AddFlags(cmd)
	return cmd
}

func listExecs(cfg *config.PlanExecList, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	var results []*models.PlanExecutionQueryResult
	var cursor *string

	for {
		params := planexecs.NewListPlanExecutionsParams().
			WithOrgName(cfg.Org)
		if cfg.PlanID != "" {
			params.WithPlanID(&cfg.PlanID)
		}
		if cfg.Tag != "" {
			params.WithTag(&cfg.Tag)
		}
		if cfg.Phase != "" {
			params.WithPhase(&cfg.Phase)
		}
		if cursor != nil {
			params.WithCursor(cursor)
		}

		resp, err := cfg.Client.PlanExecutions.ListPlanExecutions(params, nil)
		if err != nil {
			return err
		}
		results = append(results, resp.Payload...)

		// If fewer results than default page size, we're done.
		if len(resp.Payload) == 0 {
			break
		}
		last := resp.Payload[len(resp.Payload)-1]
		if last.Cursor == "" {
			break
		}
		cursor = &last.Cursor
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printExecTable(out, results)
	case config.OutputFormatJSON:
		return print.RawJSON(out, results)
	case config.OutputFormatYAML:
		return print.RawYAML(out, results)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
