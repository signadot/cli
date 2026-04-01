package plan

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
	"github.com/xeonx/timeago"
)

type planRow struct {
	ID       string `sdtab:"ID"`
	Prompt   string `sdtab:"PROMPT,trunc"`
	Steps    string `sdtab:"STEPS"`
	Created  string `sdtab:"CREATED"`
}

func printPlanTable(out io.Writer, plans []*models.RunnablePlan) error {
	t := sdtab.New[planRow](out)
	t.AddHeader()
	for _, p := range plans {
		var (
			prompt string
			steps  int
		)
		if p.Spec != nil {
			prompt = print.FirstLine(p.Spec.Prompt)
			steps = len(p.Spec.Steps)
		}
		var created string
		if p.Status != nil && p.Status.CreatedAt != "" {
			if ts, err := time.Parse(time.RFC3339, p.Status.CreatedAt); err == nil {
				created = timeago.NoMax(timeago.English).Format(ts)
			}
		}
		t.AddRow(planRow{
			ID:      p.ID,
			Prompt:  prompt,
			Steps:   fmt.Sprintf("%d", steps),
			Created: created,
		})
	}
	return t.Flush()
}

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
		fmt.Fprintf(tw, "Steps:\t%d\n", len(p.Spec.Steps))
		if len(p.Spec.Params) > 0 {
			names := make([]string, len(p.Spec.Params))
			for i, param := range p.Spec.Params {
				names[i] = param.Name
			}
			fmt.Fprintf(tw, "Params:\t%s\n", strings.Join(names, ", "))
		}
		if len(p.Spec.Output) > 0 {
			outputNames := make([]string, 0, len(p.Spec.Output))
			for k := range p.Spec.Output {
				outputNames = append(outputNames, k)
			}
			slices.Sort(outputNames)
			fmt.Fprintf(tw, "Outputs:\t%s\n", strings.Join(outputNames, ", "))
		}
		if len(p.Spec.Requires) > 0 {
			fmt.Fprintf(tw, "Requires:\t%s\n", strings.Join(p.Spec.Requires, ", "))
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

	return tw.Flush()
}
