package planshared

import (
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
)

// PrintPlanDetails renders a plan's full detail view: a top-level
// metadata header followed by the Inputs / Steps / Outputs sections.
// Used by plan get, compile, create, recompile, and run.
func PrintPlanDetails(out io.Writer, p *models.RunnablePlan) error {
	if err := printPlanHeader(out, p); err != nil {
		return err
	}
	PrintPlanBody(out, p)
	return nil
}

// PrintPlanBody renders only the Inputs / Steps / Outputs sections of
// a plan, skipping the top-level metadata header. Used when the plan
// is embedded under a parent block (e.g. plan tag get) where the
// parent already covers the plan's identity and timestamps.
func PrintPlanBody(out io.Writer, p *models.RunnablePlan) {
	if p == nil || p.Spec == nil {
		return
	}
	if len(p.Spec.Params) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Inputs:")
		printPlanParams(out, p.Spec.Params)
	}
	if len(p.Spec.Steps) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Steps:")
		printPlanSteps(out, p.Spec.Steps)
	}
	if len(p.Spec.Output) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Outputs:")
		printPlanOutputs(out, p.Spec.Output)
	}
}

func printPlanHeader(out io.Writer, p *models.RunnablePlan) error {
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	fmt.Fprintf(tw, "ID:\t%s\n", p.ID)
	if p.Spec != nil {
		if p.Spec.SelectionHint != "" {
			fmt.Fprintf(tw, "Selection Hint:\t%s\n", p.Spec.SelectionHint)
		}
		if p.Spec.Prompt != "" {
			fmt.Fprintf(tw, "Prompt:\t%s\n", print.FirstLine(p.Spec.Prompt))
		}
		if p.Spec.Runner != "" {
			fmt.Fprintf(tw, "Runner:\t%s\n", p.Spec.Runner)
		}
		if c := p.Spec.Cluster; c != nil {
			switch {
			case c.FromCluster != "":
				fmt.Fprintf(tw, "Cluster:\tfrom param %q\n", c.FromCluster)
			case c.FromSandbox != "":
				fmt.Fprintf(tw, "Cluster:\tfrom sandbox param %q\n", c.FromSandbox)
			case c.FromRouteGroup != "":
				fmt.Fprintf(tw, "Cluster:\tfrom route group param %q\n", c.FromRouteGroup)
			case c.Pattern != "":
				fmt.Fprintf(tw, "Cluster:\tpattern %q\n", c.Pattern)
			}
		}
	}
	if p.Status != nil {
		fmt.Fprintf(tw, "Created:\t%s\n", utils.FormatTimestamp(p.Status.CreatedAt))
		if by := FormatCreatedBy(p.Status.CreatedBy); by != "" {
			fmt.Fprintf(tw, "Created By:\t%s\n", by)
		}
		if p.Status.CompiledFrom != "" {
			fmt.Fprintf(tw, "Compiled From:\t%s\n", p.Status.CompiledFrom)
		}
		if p.Status.Executions > 0 {
			fmt.Fprintf(tw, "Executions:\t%d\n", p.Status.Executions)
		}
	}
	return tw.Flush()
}

// printPlanParams renders each declared param. Defaults render with
// the arrow form used elsewhere; required params with no default get
// a "(required)" tag, optional ones with no default a "(optional)".
func printPlanParams(out io.Writer, params []*models.PlanField) {
	maxName := 0
	for _, p := range params {
		if p != nil && len(p.Name) > maxName {
			maxName = len(p.Name)
		}
	}
	for _, p := range params {
		if p == nil {
			continue
		}
		var detail, via string
		switch {
		case p.Default != nil:
			detail = FormatValue(p.Default)
			via = "default"
		case p.Required:
			via = "required"
		default:
			via = "optional"
		}
		PrintInputLine(out, "  ", maxName, p.Name, detail, via)
	}
}

// Plan outputs are always refs (mappings to step outputs), so we
// drop the trailing (ref) tag — there's no other variant to
// disambiguate against.
func printPlanOutputs(out io.Writer, outputs map[string]string) {
	names := make([]string, 0, len(outputs))
	for k := range outputs {
		names = append(names, k)
	}
	sort.Strings(names)
	maxName := 0
	for _, n := range names {
		if len(n) > maxName {
			maxName = len(n)
		}
	}
	for _, n := range names {
		fmt.Fprintf(out, "  %-*s   ← %s\n", maxName, n, outputs[n])
	}
}

func printPlanSteps(out io.Writer, steps []*models.PlanStep) {
	for i, s := range steps {
		if s == nil {
			continue
		}
		if i > 0 {
			fmt.Fprintln(out)
		}
		fmt.Fprintf(out, "  %s\n", s.ID)
		if s.Action != nil {
			line := "    action: " + ActionLabel(s.Action)
			if s.Action.Revision > 0 {
				line += fmt.Sprintf("   (revision %d)", s.Action.Revision)
			}
			fmt.Fprintln(out, line)
			if img := FormatImage(s.Action.Image); img != "" {
				fmt.Fprintf(out, "    image: %s\n", img)
			}
			if s.Action.Timeout != "" {
				fmt.Fprintf(out, "    timeout: %s\n", s.Action.Timeout)
			} else if s.Action.TimeoutInputName != "" {
				fmt.Fprintf(out, "    timeout: (from input %q)\n", s.Action.TimeoutInputName)
			}
		}
		if s.Condition != "" {
			fmt.Fprintf(out, "    when: %s\n", s.Condition)
		}
		printStepInputs(out, s)
		printStepOutputs(out, s)
	}
}

// printStepInputs lists the inputs the plan author wired for this
// step. Values from step.Args.Values render as "set in plan"; refs
// from step.Args.Refs render as "ref". Inputs that were neither set
// nor wired (will fall back to defaults or be unbound at run time)
// aren't shown here; that resolution is per-execution and lives in
// plan x get.
func printStepInputs(out io.Writer, s *models.PlanStep) {
	values := ParamsAsMap(StepArgsValues(s))
	var refs map[string]string
	if s.Args != nil {
		refs = s.Args.Refs
	}
	if len(values) == 0 && len(refs) == 0 {
		return
	}

	type wired struct {
		name, detail, via string
	}
	var inputs []wired
	for name, v := range values {
		inputs = append(inputs, wired{name, FormatValue(v), "set in plan"})
	}
	for name, r := range refs {
		inputs = append(inputs, wired{name, r, "ref"})
	}
	sort.Slice(inputs, func(i, j int) bool { return inputs[i].name < inputs[j].name })

	fmt.Fprintln(out, "    inputs:")
	maxName := 0
	for _, in := range inputs {
		if len(in.name) > maxName {
			maxName = len(in.name)
		}
	}
	for _, in := range inputs {
		PrintInputLine(out, "      ", maxName, in.name, in.detail, in.via)
	}
}

// printStepOutputs lists the step's declared extra_outputs (the step-
// level extension over the action's declared outputs) with their
// schema summary. The action's own outputs are inherent to the action
// and visible via signadot plan action get.
func printStepOutputs(out io.Writer, s *models.PlanStep) {
	if len(s.ExtraOutputs) == 0 {
		return
	}
	fmt.Fprintln(out, "    outputs:")
	maxName := 0
	for _, o := range s.ExtraOutputs {
		if o != nil && len(o.Name) > maxName {
			maxName = len(o.Name)
		}
	}
	for _, o := range s.ExtraOutputs {
		if o == nil {
			continue
		}
		fmt.Fprintf(out, "      %-*s   %s\n", maxName, o.Name, formatFieldSchema(o))
	}
}

func formatFieldSchema(f *models.PlanField) string {
	if f.SchemaRef != "" {
		return fmt.Sprintf("schema: %s", f.SchemaRef)
	}
	if m, ok := f.Schema.(map[string]any); ok {
		if t, ok := m["type"].(string); ok && t != "" {
			return fmt.Sprintf("schema: %s", t)
		}
	}
	return ""
}
