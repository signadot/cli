package planexec

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/docker/go-units"
	"github.com/signadot/cli/internal/command/planshared"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
)

// PrintRunResult is a thin wrapper around printExecDetails for the
// plan run path. The output-rendering logic that used to live here
// now lives in printExecDetails so plan x get and plan x cancel see
// the same trailing inline values + artifact outputs section.
func PrintRunResult(out io.Writer, ex *models.PlanExecution, planSpec *models.PlanSpec) error {
	return printExecDetails(out, ex, planSpec)
}

func printExecDetails(out io.Writer, ex *models.PlanExecution, planSpec *models.PlanSpec) error {
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)

	fmt.Fprintf(tw, "ID:\t%s\n", ex.ID)
	if ex.Spec != nil {
		fmt.Fprintf(tw, "Plan:\t%s\n", ex.Spec.PlanID)
		if ex.Spec.Cluster != "" {
			fmt.Fprintf(tw, "Cluster:\t%s\n", ex.Spec.Cluster)
		}
		if ex.Spec.Runner != "" {
			fmt.Fprintf(tw, "Runner:\t%s\n", ex.Spec.Runner)
		}
	}
	if ex.Status != nil {
		fmt.Fprintf(tw, "Phase:\t%s\n", ex.Status.Phase)
		fmt.Fprintf(tw, "Created:\t%s\n", utils.FormatTimestamp(ex.Status.CreatedAt))
		if ex.Status.UpdatedAt != "" {
			fmt.Fprintf(tw, "Updated:\t%s\n", utils.FormatTimestamp(ex.Status.UpdatedAt))
		}
		if ex.Status.CompletedAt != "" {
			fmt.Fprintf(tw, "Completed:\t%s\n", utils.FormatTimestamp(ex.Status.CompletedAt))
		}
		if sc := ex.Status.StepCounts; sc != nil {
			total := sc.Init + sc.Waiting + sc.Running + sc.Completed + sc.Failed + sc.Skipped
			fmt.Fprintf(tw, "Steps:\t%d/%d completed", sc.Completed, total)
			if sc.Failed > 0 {
				fmt.Fprintf(tw, ", %d failed", sc.Failed)
			}
			if sc.Running > 0 {
				fmt.Fprintf(tw, ", %d running", sc.Running)
			}
			fmt.Fprintln(tw)
		}
		if ex.Status.Error != "" {
			fmt.Fprintf(tw, "Error:\t%s\n", ex.Status.Error)
		}
	}

	if err := tw.Flush(); err != nil {
		return err
	}

	if ex.Status != nil && len(ex.Status.Inputs) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Inputs:")
		var supplied any
		if ex.Spec != nil {
			supplied = ex.Spec.Params
		}
		printPlanInputs(out, ex.Status.Inputs, supplied, planSpec)
	}

	if ex.Status != nil && len(ex.Status.Steps) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Steps:")
		printSteps(out, ex.Status.Steps, planSpec)
	}

	printExecOutputs(out, ex)
	return nil
}

// printExecOutputs renders an execution's plan-level outputs in the
// indented arrow form used by the rest of the detail view. Inline
// values get the arrow + truncated value; artifacts show kind/size.
// Each row is annotated with the source step (StepRef) when set.
// Skipped for executions with no plan-level outputs.
func printExecOutputs(out io.Writer, ex *models.PlanExecution) {
	if ex.Status == nil || len(ex.Status.Outputs) == 0 {
		return
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Outputs:")
	maxName := 0
	for _, o := range ex.Status.Outputs {
		if o != nil && len(o.Name) > maxName {
			maxName = len(o.Name)
		}
	}
	for _, o := range ex.Status.Outputs {
		if o == nil {
			continue
		}
		stepID := ""
		if o.StepRef != nil {
			stepID = o.StepRef.StepID
		}
		printOutputRow(out, "  ", maxName, o.Name, o.Value, o.Artifact, o.Metadata, stepID)
	}
}

// printOutputRow emits one output entry. Inline values use the arrow
// form; artifacts have no inline value, just the kind/size tag.
// stepID, when non-empty, is appended to the tag as ", from <stepID>"
// — used for plan-level outputs to surface which step produced them.
func printOutputRow(out io.Writer, indent string, nameWidth int, name string, value any, art *models.PlanArtifactRef, metadata map[string]string, stepID string) {
	detail := ""
	if value != nil {
		detail = planshared.FormatValue(value)
	}
	planshared.PrintInputLine(out, indent, nameWidth, name, detail, formatOutputTag(art, metadata, stepID))
}

// formatOutputTag renders the parenthesized tag content for an
// output:
//
//	inline                      — for inline values
//	artifact, <size>, ready     — for artifacts (ready state always shown)
//	artifact, <size>, not ready
//
// When the output's metadata carries a "contentType" key it's
// appended after the ready flag. A non-empty stepID adds a trailing
// ", from <stepID>" for plan-level rollups.
func formatOutputTag(art *models.PlanArtifactRef, metadata map[string]string, stepID string) string {
	var parts []string
	if art != nil {
		parts = append(parts, "artifact")
		if art.Size > 0 {
			parts = append(parts, units.HumanSize(float64(art.Size)))
		}
		if art.Ready {
			parts = append(parts, "ready")
		} else {
			parts = append(parts, "not ready")
		}
		if ct := metadata["contentType"]; ct != "" {
			parts = append(parts, ct)
		}
		if art.Error != "" {
			parts = append(parts, "error: "+art.Error)
		}
	} else {
		parts = append(parts, "inline")
	}
	if stepID != "" {
		parts = append(parts, "from "+stepID)
	}
	return strings.Join(parts, ", ")
}

// printPlanInputs prints plan-level params in the indented arrow form
// used throughout the steps section. All entries are shown so the
// reader sees the authoritative view of what bound at dispatch time.
// suppliedParams is execution.spec.params (the raw any from the
// SDK); default values are pulled from the plan spec when it's
// available.
func printPlanInputs(out io.Writer, inputs []*models.PlanInputStatus, suppliedParams any, planSpec *models.PlanSpec) {
	maxName := 0
	for _, i := range inputs {
		if len(i.Name) > maxName {
			maxName = len(i.Name)
		}
	}
	supplied := planshared.ParamsAsMap(suppliedParams)
	for _, i := range inputs {
		detail := ""
		switch i.ResolvedVia {
		case models.PlansInputResolvedViaCallerSecret:
			detail = i.SecretName
		case models.PlansInputResolvedViaLiteral:
			if v, ok := supplied[i.Name]; ok {
				detail = planshared.FormatValue(v)
			}
		case models.PlansInputResolvedViaDefault:
			if p := findPlanParam(planSpec, i.Name); p != nil && p.Default != nil {
				detail = planshared.FormatValue(p.Default)
			}
		}
		planshared.PrintInputLine(out, "  ", maxName, i.Name, detail, planLevelVia(i.ResolvedVia))
	}
}

// planLevelVia renders a plan-level input's resolution method as
// human-readable prose. The plain enum names (literal, default, ...)
// are ambiguous at this level: "literal" really means "the caller
// passed --param" and "default" means "the plan's declared default
// kicked in".
func planLevelVia(via models.PlansInputResolvedVia) string {
	switch via {
	case models.PlansInputResolvedViaLiteral:
		return "from --param"
	case models.PlansInputResolvedViaCallerSecret:
		return "from --param-secret"
	case models.PlansInputResolvedViaDefault:
		return "plan default"
	case models.PlansInputResolvedViaUnbound:
		return "unbound"
	}
	return string(via)
}

// stepLevelVia renders a step-level input's resolution method as
// human-readable prose. "literal" at this level means the plan author
// wrote the value directly into step args (rather than the caller
// supplying it), so we say "set in plan".
func stepLevelVia(via models.PlansInputResolvedVia) string {
	switch via {
	case models.PlansInputResolvedViaLiteral:
		return "set in plan"
	case models.PlansInputResolvedViaRef:
		return "ref"
	case models.PlansInputResolvedViaDefault:
		return "default"
	case models.PlansInputResolvedViaUnbound:
		return "unbound"
	}
	return string(via)
}

func printSteps(out io.Writer, steps []*models.PlanStepStatus, planSpec *models.PlanSpec) {
	maxID := 0
	for _, s := range steps {
		if len(s.ID) > maxID {
			maxID = len(s.ID)
		}
	}
	for i, s := range steps {
		step := findStep(planSpec, s.ID)
		hasAction := step != nil && step.Action != nil
		hasBody := hasStepBody(s) || hasAction
		if i > 0 && hasBody {
			fmt.Fprintln(out)
		}
		fmt.Fprintf(out, "  %-*s   %s\n", maxID, s.ID, s.Phase)
		if hasAction {
			label := planshared.ActionLabel(step.Action)
			if step.Action.Revision > 0 {
				label += fmt.Sprintf("   (revision %d)", step.Action.Revision)
			}
			fmt.Fprintf(out, "    action: %s\n", label)
		}
		if s.Error != "" {
			fmt.Fprintf(out, "    error: %s\n", s.Error)
		}
		if len(s.Inputs) > 0 {
			fmt.Fprintln(out, "    inputs:")
			stepValues := planshared.ParamsAsMap(planshared.StepArgsValues(step))
			maxName := 0
			for _, in := range s.Inputs {
				if len(in.Name) > maxName {
					maxName = len(in.Name)
				}
			}
			for _, in := range s.Inputs {
				detail := ""
				switch in.ResolvedVia {
				case models.PlansInputResolvedViaRef:
					detail = in.Ref
				case models.PlansInputResolvedViaLiteral:
					if v, ok := stepValues[in.Name]; ok {
						detail = planshared.FormatValue(v)
					}
				case models.PlansInputResolvedViaDefault:
					if p := findStepInputDecl(step, in.Name); p != nil && p.Default != nil {
						detail = planshared.FormatValue(p.Default)
					}
				}
				planshared.PrintInputLine(out, "      ", maxName, in.Name, detail, stepLevelVia(in.ResolvedVia))
			}
		}
		if len(s.Outputs) > 0 {
			fmt.Fprintln(out, "    outputs:")
			maxName := 0
			for _, o := range s.Outputs {
				if o != nil && len(o.Name) > maxName {
					maxName = len(o.Name)
				}
			}
			for _, o := range s.Outputs {
				if o == nil {
					continue
				}
				printOutputRow(out, "      ", maxName, o.Name, o.Value, o.Artifact, o.Metadata, "")
			}
		}
	}
}

func hasStepBody(s *models.PlanStepStatus) bool {
	return len(s.Inputs) > 0 || len(s.Outputs) > 0 || s.Error != ""
}


func findPlanParam(planSpec *models.PlanSpec, name string) *models.PlanField {
	if planSpec == nil {
		return nil
	}
	for _, p := range planSpec.Params {
		if p != nil && p.Name == name {
			return p
		}
	}
	return nil
}

func findStep(planSpec *models.PlanSpec, stepID string) *models.PlanStep {
	if planSpec == nil {
		return nil
	}
	for _, s := range planSpec.Steps {
		if s != nil && s.ID == stepID {
			return s
		}
	}
	return nil
}

// findStepInputDecl looks up a step input by name, searching the
// action's declared params first and falling back to the step's
// declared extra_inputs. Either can carry a Default the runner
// resolved against.
func findStepInputDecl(step *models.PlanStep, name string) *models.PlanField {
	if step == nil {
		return nil
	}
	if step.Action != nil {
		for _, p := range step.Action.Params {
			if p != nil && p.Name == name {
				return p
			}
		}
	}
	for _, p := range step.ExtraInputs {
		if p != nil && p.Name == name {
			return p
		}
	}
	return nil
}

type allOutputRow struct {
	Name    string `sdtab:"NAME"`
	Step    string `sdtab:"STEP"`
	Scope   string `sdtab:"SCOPE"`
	Storage string `sdtab:"STORAGE"`
	Size    string `sdtab:"SIZE"`
	Ready   string `sdtab:"READY"`
}

func printAllOutputsTable(out io.Writer, outputs []allOutput) error {
	t := sdtab.New[allOutputRow](out)
	t.AddHeader()
	for _, o := range outputs {
		size := ""
		ready := "-"
		if o.Size > 0 {
			size = units.HumanSize(float64(o.Size))
		}
		if o.Ready != nil {
			if *o.Ready {
				ready = "true"
			} else {
				ready = "false"
			}
		}
		t.AddRow(allOutputRow{
			Name:    o.Name,
			Step:    o.Step,
			Scope:   o.Scope,
			Storage: o.Type,
			Size:    size,
			Ready:   ready,
		})
	}
	return t.Flush()
}

type execRow struct {
	ID      string `sdtab:"ID"`
	Plan    string `sdtab:"PLAN"`
	Phase   string `sdtab:"PHASE"`
	Steps   string `sdtab:"STEPS"`
	Created string `sdtab:"CREATED"`
}

func printExecTable(out io.Writer, results []*models.PlanExecutionQueryResult) error {
	t := sdtab.New[execRow](out)
	t.AddHeader()
	for _, r := range results {
		var plan, phase, steps, created string
		if r.Spec != nil {
			plan = r.Spec.PlanID
		}
		if r.Status != nil {
			phase = string(r.Status.Phase)
			if sc := r.Status.StepCounts; sc != nil {
				total := sc.Init + sc.Waiting + sc.Running + sc.Completed + sc.Failed + sc.Skipped
				steps = fmt.Sprintf("%d/%d", sc.Completed, total)
			}
			created = utils.TimeAgo(r.Status.CreatedAt)
		}
		t.AddRow(execRow{
			ID:      r.ID,
			Plan:    plan,
			Phase:   phase,
			Steps:   steps,
			Created: created,
		})
	}
	return t.Flush()
}
