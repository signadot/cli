package planexec

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/docker/go-units"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
)

// PrintRunResult prints execution details with output summary for plan run.
// Inline values are printed to stdout; artifact outputs are listed by reference.
// planSpec is the compiled plan being executed; when non-nil it is used to
// resolve literal/default input values into the per-step detail. Pass nil
// when the plan spec isn't available (e.g. the plan was deleted).
func PrintRunResult(out io.Writer, ex *models.PlanExecution, planSpec *models.PlanSpec) error {
	if err := printExecDetails(out, ex, planSpec); err != nil {
		return err
	}

	if ex.Status == nil || len(ex.Status.Outputs) == 0 {
		return nil
	}

	// Print inline values directly.
	for _, o := range ex.Status.Outputs {
		if o.Value != nil {
			fmt.Fprintf(out, "\n--- %s ---\n%v\n", o.Name, o.Value)
		}
	}

	// List artifact outputs by reference.
	var artifacts []*models.PlanOutputStatus
	for _, o := range ex.Status.Outputs {
		if o.Artifact != nil {
			artifacts = append(artifacts, o)
		}
	}
	if len(artifacts) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Artifact outputs:")
		return printOutputsTable(out, artifacts)
	}
	return nil
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
		printPlanInputs(out, ex, planSpec)
	}

	if ex.Status != nil && len(ex.Status.Steps) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Steps:")
		printSteps(out, ex.Status.Steps, planSpec)
	}

	return nil
}

// printPlanInputs prints plan-level params in the indented arrow form
// used throughout the steps section. All entries are shown so the
// reader sees the authoritative view of what bound at dispatch time.
// Literal values are taken from execution.spec.params; default values
// are pulled from the plan spec when it's available.
func printPlanInputs(out io.Writer, ex *models.PlanExecution, planSpec *models.PlanSpec) {
	inputs := ex.Status.Inputs
	maxName := 0
	for _, i := range inputs {
		if len(i.Name) > maxName {
			maxName = len(i.Name)
		}
	}
	suppliedParams := paramsAsMap(execParams(ex))
	for _, i := range inputs {
		detail := ""
		switch i.ResolvedVia {
		case models.PlansInputResolvedViaCallerSecret:
			detail = i.SecretName
		case models.PlansInputResolvedViaLiteral:
			if v, ok := suppliedParams[i.Name]; ok {
				detail = formatValue(v)
			}
		case models.PlansInputResolvedViaDefault:
			if p := findPlanParam(planSpec, i.Name); p != nil && p.Default != nil {
				detail = formatValue(p.Default)
			}
		}
		printInputLine(out, "  ", maxName, i.Name, detail, planLevelVia(i.ResolvedVia))
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
			label := actionLabel(step.Action)
			if step.Action.Revision > 0 {
				label += fmt.Sprintf("   (revision %d)", step.Action.Revision)
			}
			fmt.Fprintf(out, "    action: %s\n", label)
		}
		if s.Error != "" {
			fmt.Fprintf(out, "    error: %s\n", s.Error)
		}
		if len(s.Inputs) == 0 {
			continue
		}
		fmt.Fprintln(out, "    inputs:")
		stepValues := paramsAsMap(stepArgsValues(step))
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
					detail = formatValue(v)
				}
			case models.PlansInputResolvedViaDefault:
				if p := findStepInputDecl(step, in.Name); p != nil && p.Default != nil {
					detail = formatValue(p.Default)
				}
			}
			printInputLine(out, "      ", maxName, in.Name, detail, stepLevelVia(in.ResolvedVia))
		}
	}
}

func hasStepBody(s *models.PlanStepStatus) bool {
	return len(s.Inputs) > 0 || s.Error != ""
}

// actionLabel renders the step's action as "name (id)" when the
// server returns both, name alone when the ID is empty, or ID alone
// when the name isn't populated yet (older plans compiled before the
// Name field was added).
func actionLabel(a *models.PlanStepAction) string {
	if a == nil {
		return ""
	}
	if a.Name == "" {
		return a.ActionID
	}
	if a.ActionID == "" {
		return a.Name
	}
	return fmt.Sprintf("%s (%s)", a.Name, a.ActionID)
}

// printInputLine emits one input row in the form
//
//	<indent><name padded>   ← <detail>   (<via>)
//
// or, when there's no detail to show:
//
//	<indent><name padded>   (<via>)
//
// Manual padding is used (rather than tabwriter) because rows with
// and without a detail cell have different "shapes"; a single
// tabwriter scope would size the empty middle column based on the
// longest detail and leave the (via) tag for no-detail rows pushed
// far to the right.
func printInputLine(out io.Writer, indent string, nameWidth int, name, detail, via string) {
	if detail == "" {
		fmt.Fprintf(out, "%s%-*s   (%s)\n", indent, nameWidth, name, via)
		return
	}
	fmt.Fprintf(out, "%s%-*s   ← %s   (%s)\n", indent, nameWidth, name, detail, via)
}

// formatValue renders a parameter value for a detail column. Strings
// pass through as bare text; everything else round-trips through JSON
// so objects/arrays stay copy-pasteable into a plan spec rather than
// rendering as Go's map[a:1] / [1 2] forms.
func formatValue(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

func execParams(ex *models.PlanExecution) any {
	if ex == nil || ex.Spec == nil {
		return nil
	}
	return ex.Spec.Params
}

func stepArgsValues(step *models.PlanStep) any {
	if step == nil || step.Args == nil {
		return nil
	}
	return step.Args.Values
}

func paramsAsMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
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

type outputRow struct {
	Name  string `sdtab:"NAME"`
	Step  string `sdtab:"STEP"`
	Type  string `sdtab:"TYPE"`
	Size  string `sdtab:"SIZE"`
	Ready string `sdtab:"READY"`
}

func printOutputsTable(out io.Writer, outputs []*models.PlanOutputStatus) error {
	t := sdtab.New[outputRow](out)
	t.AddHeader()
	for _, o := range outputs {
		step := ""
		if o.StepRef != nil {
			step = o.StepRef.StepID
		}
		typ := "inline"
		size := ""
		ready := "-"
		if o.Artifact != nil {
			typ = "artifact"
			size = units.HumanSize(float64(o.Artifact.Size))
			if o.Artifact.Ready {
				ready = "true"
			} else {
				ready = "false"
			}
		}
		t.AddRow(outputRow{
			Name:  o.Name,
			Step:  step,
			Type:  typ,
			Size:  size,
			Ready: ready,
		})
	}
	return t.Flush()
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
