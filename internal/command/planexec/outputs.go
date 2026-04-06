package planexec

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	planexecs "github.com/signadot/go-sdk/client/plan_executions"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newOutputs(exec *config.PlanExecution) *cobra.Command {
	cfg := &config.PlanExecOutputs{PlanExecution: exec}

	cmd := &cobra.Command{
		Use:   "outputs EXECUTION_ID",
		Short: "List all outputs of a plan execution (plan-level and step-level)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return listOutputs(cfg, cmd.OutOrStdout(), args[0])
		},
	}

	return cmd
}

// allOutput unifies plan-level and step-level outputs for display.
type allOutput struct {
	Name  string `json:"name"`
	Step  string `json:"step"`
	Scope string `json:"scope"` // "plan" or "step"
	Type  string `json:"type"`  // "inline" or "artifact"
	Size  int64  `json:"size,omitempty"`
	Ready *bool  `json:"ready,omitempty"`
}

func listOutputs(cfg *config.PlanExecOutputs, out io.Writer, execID string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	params := planexecs.NewGetPlanExecutionParams().
		WithOrgName(cfg.Org).
		WithExecutionID(execID)
	resp, err := cfg.Client.PlanExecutions.GetPlanExecution(params, nil)
	if err != nil {
		return err
	}

	all := collectAllOutputs(resp.Payload)

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printAllOutputsTable(out, all)
	case config.OutputFormatJSON:
		return print.RawJSON(out, all)
	case config.OutputFormatYAML:
		return print.RawYAML(out, all)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}

func collectAllOutputs(ex *models.PlanExecution) []allOutput {
	if ex.Status == nil {
		return nil
	}

	// Track plan-level output names to avoid duplicating them in step section.
	planOutputNames := map[string]bool{}
	var all []allOutput

	// Plan-level outputs.
	for _, o := range ex.Status.Outputs {
		step := ""
		if o.StepRef != nil {
			step = o.StepRef.StepID
		}
		planOutputNames[step+"/"+o.Name] = true
		all = append(all, allOutput{
			Name:  o.Name,
			Step:  step,
			Scope: "plan",
			Type:  outputType(o.Artifact),
			Size:  outputSize(o.Artifact, o.Value),
			Ready: outputReady(o.Artifact),
		})
	}

	// Step-level outputs (skip those already shown as plan-level).
	for _, s := range ex.Status.Steps {
		for _, o := range s.Outputs {
			key := s.ID + "/" + o.Name
			if planOutputNames[key] {
				continue
			}
			all = append(all, allOutput{
				Name:  o.Name,
				Step:  s.ID,
				Scope: "step",
				Type:  outputType(o.Artifact),
				Size:  outputSize(o.Artifact, o.Value),
				Ready: outputReady(o.Artifact),
			})
		}
	}

	return all
}

func outputType(a *models.PlanArtifactRef) string {
	if a != nil {
		return "artifact"
	}
	return "inline"
}

func outputSize(a *models.PlanArtifactRef, value any) int64 {
	if a != nil {
		return a.Size
	}
	if value != nil {
		if s, ok := value.(string); ok {
			return int64(len(s))
		}
		b, err := json.Marshal(value)
		if err == nil {
			return int64(len(b))
		}
	}
	return 0
}

func outputReady(a *models.PlanArtifactRef) *bool {
	t := true
	if a != nil {
		return &a.Ready
	}
	return &t
}
