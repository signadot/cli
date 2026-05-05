package plan

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
)

func printPlanDetails(out io.Writer, p *models.RunnablePlan) error {
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)

	fmt.Fprintf(tw, "ID:\t%s\n", p.ID)
	if p.Spec != nil {
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
		if p.Status.CompiledFrom != "" {
			fmt.Fprintf(tw, "Compiled From:\t%s\n", p.Status.CompiledFrom)
		}
		if p.Status.Executions > 0 {
			fmt.Fprintf(tw, "Executions:\t%d\n", p.Status.Executions)
		}
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	if p.Spec == nil {
		return nil
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

	return nil
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
			detail = formatValue(p.Default)
			via = "default"
		case p.Required:
			via = "required"
		default:
			via = "optional"
		}
		printInputLine(out, "  ", maxName, p.Name, detail, via)
	}
}

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
	// Plan outputs are always refs (mappings to step outputs), so we
	// drop the trailing (ref) tag — there's no other variant to
	// disambiguate against.
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
			line := "    action: " + actionLabel(s.Action)
			if s.Action.Revision > 0 {
				line += fmt.Sprintf("   (revision %d)", s.Action.Revision)
			}
			fmt.Fprintln(out, line)
			if img := formatImage(s.Action.Image); img != "" {
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
	values := paramsAsMap(stepArgsValues(s))
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
		inputs = append(inputs, wired{name, formatValue(v), "set in plan"})
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
		printInputLine(out, "      ", maxName, in.name, in.detail, in.via)
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
		schema := formatFieldSchema(o)
		fmt.Fprintf(out, "      %-*s   %s\n", maxName, o.Name, schema)
	}
}

// printInputLine emits one input row in the form
//
//	<indent><name padded>   ← <detail>   (<via>)
//
// or, when there's no detail to show:
//
//	<indent><name padded>   (<via>)
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
// rendering as Go's map[a:1] / [1 2] forms. Multi-line and very long
// values are summarised so they don't break the inline arrow form;
// users who want the full value can use -o yaml.
func formatValue(v any) string {
	if v == nil {
		return ""
	}
	var raw string
	if s, ok := v.(string); ok {
		raw = s
	} else {
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		raw = string(b)
	}
	return truncateForDisplay(raw)
}

// maxValueDisplay caps single-line value rendering.
const maxValueDisplay = 80

// truncateForDisplay collapses a value into a single line suitable
// for the inline arrow form. Multi-line values are reduced to the
// first line + a (N lines) marker; over-long single lines get a
// trailing ellipsis.
func truncateForDisplay(s string) string {
	trimmed := strings.TrimRight(s, "\n")
	if firstLine, _, multiline := strings.Cut(trimmed, "\n"); multiline {
		nLines := strings.Count(trimmed, "\n") + 1
		if len(firstLine) > maxValueDisplay {
			firstLine = firstLine[:maxValueDisplay]
		}
		return fmt.Sprintf("%s… (%d lines)", firstLine, nLines)
	}
	if len(trimmed) > maxValueDisplay {
		return trimmed[:maxValueDisplay] + "…"
	}
	return trimmed
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

func formatImage(ref *models.PlanImageRef) string {
	if ref == nil {
		return ""
	}
	if ref.Literal != "" {
		return ref.Literal
	}
	if ref.InputName != "" {
		return fmt.Sprintf("(from input %q)", ref.InputName)
	}
	return ""
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

func stepArgsValues(s *models.PlanStep) any {
	if s == nil || s.Args == nil {
		return nil
	}
	return s.Args.Values
}

func paramsAsMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}
