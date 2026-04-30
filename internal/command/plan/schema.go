package plan

import (
	"encoding/json"
	"io"

	"github.com/signadot/cli/internal/config"
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
	b, err := json.Marshal(resp.Payload)
	if err != nil {
		return err
	}
	if _, err := out.Write(b); err != nil {
		return err
	}
	_, err = out.Write([]byte("\n"))
	return err
}
