package plan

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/signadot/cli/internal/command/plantag"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/jsonexact"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/utils"
	sdkplans "github.com/signadot/go-sdk/client/plans"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newCreate(plan *config.Plan) *cobra.Command {
	cfg := &config.PlanCreate{Plan: plan}

	cmd := &cobra.Command{
		Use:   "create -f SPEC_FILE",
		Short: "Create a plan from a hand-authored spec file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return create(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}

	cfg.AddFlags(cmd)
	return cmd
}

func create(cfg *config.PlanCreate, out, log io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	spec, err := loadPlanSpec(cfg.Filename, cfg.TemplateVals)
	if err != nil {
		return err
	}

	params := sdkplans.NewCreatePlanParams().
		WithOrgName(cfg.Org).
		WithData(spec)
	resp, err := cfg.Client.Plans.CreatePlan(params, nil)
	if err != nil {
		return err
	}

	if cfg.Tag != "" {
		if _, err := plantag.ApplyTag(cfg.Plan, resp.Payload.ID, cfg.Tag); err != nil {
			return fmt.Errorf("plan created (id=%s) but tagging failed: %w", resp.Payload.ID, err)
		}
		fmt.Fprintf(log, "Tagged plan %s as %q\n", resp.Payload.ID, cfg.Tag)
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printPlanDetails(out, resp.Payload)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}

func loadPlanSpec(file string, tplVals config.TemplateVals) (*models.PlanSpec, error) {
	template, err := utils.LoadUnstructuredTemplate(file, tplVals, false)
	if err != nil {
		return nil, err
	}

	// Extract the spec field if present, otherwise treat the whole thing as spec.
	m, ok := template.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("plan file must be a YAML/JSON object")
	}
	specVal, hasSpec := m["spec"]
	if hasSpec {
		template = specVal
	}

	d, err := json.Marshal(template)
	if err != nil {
		return nil, err
	}
	spec := &models.PlanSpec{}
	if err := jsonexact.Unmarshal(d, spec); err != nil {
		return nil, fmt.Errorf("couldn't parse plan spec - %s",
			strings.TrimPrefix(err.Error(), "json: "))
	}
	return spec, nil
}
