// Package planshared holds the rendering helpers used by the plan,
// planaction, planexec, and plantag commands. The helpers render
// inputs/outputs in a single arrow form and lean on the same value
// truncation rules so the four detail views stay visually in sync.
package planshared

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/signadot/go-sdk/models"
)

// MaxValueDisplay caps single-line value rendering. Keeps room on a
// 100-column terminal for the indent + name + arrow + via tag
// surrounding the value.
const MaxValueDisplay = 80

// PrintInputLine emits one row in the form
//
//	<indent><name padded>   ← <detail>   (<via>)
//
// or, when there's no detail to show:
//
//	<indent><name padded>   (<via>)
//
// Manual padding is used (rather than tabwriter) because rows with
// and without a detail cell have different shapes; a single
// tabwriter scope would size the empty middle column based on the
// longest detail and leave the (via) tag for no-detail rows pushed
// far to the right.
func PrintInputLine(out io.Writer, indent string, nameWidth int, name, detail, via string) {
	if detail == "" {
		fmt.Fprintf(out, "%s%-*s   (%s)\n", indent, nameWidth, name, via)
		return
	}
	fmt.Fprintf(out, "%s%-*s   ← %s   (%s)\n", indent, nameWidth, name, detail, via)
}

// FormatValue renders a parameter value for a detail column. Strings
// pass through as bare text; everything else round-trips through JSON
// so objects/arrays stay copy-pasteable into a plan spec rather than
// rendering as Go's map[a:1] / [1 2] forms. Multi-line and very long
// values are summarised so they don't break the inline arrow form;
// users who want the full value can use -o yaml.
func FormatValue(v any) string {
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

// truncateForDisplay collapses a value into a single line suitable
// for the inline arrow form. Multi-line values are reduced to the
// first line + a (N lines) marker; over-long single lines get a
// trailing ellipsis.
func truncateForDisplay(s string) string {
	trimmed := strings.TrimRight(s, "\n")
	if firstLine, _, multiline := strings.Cut(trimmed, "\n"); multiline {
		nLines := strings.Count(trimmed, "\n") + 1
		if len(firstLine) > MaxValueDisplay {
			firstLine = firstLine[:MaxValueDisplay]
		}
		return fmt.Sprintf("%s… (%d lines)", firstLine, nLines)
	}
	if len(trimmed) > MaxValueDisplay {
		return trimmed[:MaxValueDisplay] + "…"
	}
	return trimmed
}

// FormatImage renders a PlanImageRef for the "image:" line. Returns
// the literal image when set, "(from input \"name\")" when the image
// is bound to an input, or empty when nothing is declared.
func FormatImage(ref *models.PlanImageRef) string {
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

// ActionLabel renders the step's action as "name (id)" when the
// server returns both, name alone when the ID is empty, or ID alone
// when the name isn't populated yet (older plans compiled before the
// Name field was added).
func ActionLabel(a *models.PlanStepAction) string {
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

// ParamsAsMap returns the typical map[string]any shape of a
// PlanArgs.Values / PlanExecutionSpec.Params field, or nil if the
// untyped any is missing or shaped differently. Callers use it to
// look up step or plan-level literal values by name.
func ParamsAsMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

// StepArgsValues returns the literal values map declared on a step,
// nil-safe through both the step and its Args.
func StepArgsValues(s *models.PlanStep) any {
	if s == nil || s.Args == nil {
		return nil
	}
	return s.Args.Values
}
