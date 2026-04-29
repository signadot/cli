package planexec

import (
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
func PrintRunResult(out io.Writer, ex *models.PlanExecution) error {
	if err := printExecDetails(out, ex); err != nil {
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

func printExecDetails(out io.Writer, ex *models.PlanExecution) error {
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
		printPlanInputs(out, ex.Status.Inputs)
	}

	if ex.Status != nil && len(ex.Status.Steps) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Steps:")
		printSteps(out, ex.Status.Steps)
	}

	return nil
}

// isInterestingInput reports whether an input's resolution mechanism is
// worth showing in the per-step detail. Pure literal/default/unbound
// resolutions are noise; refs and secret-backed inputs are the
// actionable cases. Unknown enum values are treated as interesting so
// a server-side schema addition surfaces rather than getting silently
// filtered.
func isInterestingInput(via models.PlansInputResolvedVia) bool {
	switch via {
	case models.PlansInputResolvedViaLiteral,
		models.PlansInputResolvedViaDefault,
		models.PlansInputResolvedViaUnbound:
		return false
	}
	return true
}

// printPlanInputs prints plan-level params in the indented arrow form
// used throughout the steps section. All plan-level inputs are shown
// (including literal/default/unbound) since this is the authoritative
// view of what bound at dispatch time.
func printPlanInputs(out io.Writer, inputs []*models.PlanInputStatus) {
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	for _, i := range inputs {
		detail := ""
		if i.SecretName != "" {
			detail = i.SecretName
		}
		printInputLine(tw, "  ", i.Name, detail, string(i.ResolvedVia))
	}
	tw.Flush()
}

func printSteps(out io.Writer, steps []*models.PlanStepStatus) {
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	for _, s := range steps {
		fmt.Fprintf(tw, "  %s\t%s\n", s.ID, s.Phase)
		if s.Error != "" {
			fmt.Fprintf(tw, "      error:\t%s\n", s.Error)
		}
		for _, i := range s.Inputs {
			if !isInterestingInput(i.ResolvedVia) {
				continue
			}
			printInputLine(tw, "      ", i.Name, i.Ref, string(i.ResolvedVia))
		}
	}
	tw.Flush()
}

// printInputLine emits one input row in the form
//
//	<indent><name>  ←  <detail>  (<via>)
//
// or, when there's no detail to show:
//
//	<indent><name>  (<via>)
//
// Tab stops keep the arrows aligned within a single tabwriter scope.
func printInputLine(tw io.Writer, indent, name, detail, via string) {
	if detail == "" {
		fmt.Fprintf(tw, "%s%s\t\t(%s)\n", indent, name, via)
		return
	}
	fmt.Fprintf(tw, "%s%s\t← %s\t(%s)\n", indent, name, detail, via)
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
