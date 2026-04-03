package planexec

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/docker/go-units"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
	"github.com/xeonx/timeago"
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

	// Print step status table if steps are present.
	if ex.Status != nil && len(ex.Status.Steps) > 0 {
		fmt.Fprintln(out)
		return printStepTable(out, ex.Status.Steps)
	}

	return nil
}

type stepRow struct {
	ID    string `sdtab:"STEP"`
	Phase string `sdtab:"PHASE"`
	Error string `sdtab:"ERROR,trunc"`
}

func printStepTable(out io.Writer, steps []*models.PlanStepStatus) error {
	t := sdtab.New[stepRow](out)
	t.AddHeader()
	for _, s := range steps {
		t.AddRow(stepRow{
			ID:    s.ID,
			Phase: string(s.Phase),
			Error: s.Error,
		})
	}
	return t.Flush()
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
			Size:  size,
			Ready: ready,
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
			if r.Status.CreatedAt != "" {
				if ts, err := time.Parse(time.RFC3339, r.Status.CreatedAt); err == nil {
					created = timeago.NoMax(timeago.English).Format(ts)
				}
			}
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
