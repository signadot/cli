package plan

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	sdkmeta "github.com/signadot/go-sdk/client/meta"
	"github.com/spf13/cobra"
)

func newSchema(plan *config.Plan) *cobra.Command {
	cfg := &config.PlanSchema{Plan: plan}

	cmd := &cobra.Command{
		Use:    "schema",
		Short:  "Print the plan-authoring JSON Schema (machine-readable)",
		Args:   cobra.NoArgs,
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSchema(cfg, cmd.OutOrStdout())
		},
	}

	return cmd
}

func runSchema(cfg *config.PlanSchema, out io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	resp, err := cfg.Client.Meta.MetaPlans(sdkmeta.NewMetaPlansParams())
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault, config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
