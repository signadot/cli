package plantag

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/signadot/cli/internal/command/planshared"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
)

type tagRow struct {
	Name    string `sdtab:"NAME"`
	PlanID  string `sdtab:"PLAN ID"`
	Steps   string `sdtab:"STEPS"`
	Hint    string `sdtab:"SELECTION HINT"`
	Created string `sdtab:"CREATED"`
	Updated string `sdtab:"UPDATED"`
}

func printTagTable(out io.Writer, tags []*models.PlanTag) error {
	t := sdtab.New[tagRow](out)
	t.AddHeader()
	for _, tag := range tags {
		var planID, steps, hint, created, updated string
		if tag.Spec != nil {
			planID = tag.Spec.PlanID
		}
		if tag.Plan != nil && tag.Plan.Spec != nil {
			steps = fmt.Sprintf("%d", len(tag.Plan.Spec.Steps))
			hint = print.FirstLine(tag.Plan.Spec.SelectionHint)
		}
		if tag.Status != nil {
			created = utils.TimeAgo(tag.Status.CreatedAt)
			updated = utils.TimeAgo(tag.Status.UpdatedAt)
		}
		t.AddRow(tagRow{
			Name:    tag.Name,
			PlanID:  planID,
			Steps:   steps,
			Hint:    hint,
			Created: created,
			Updated: updated,
		})
	}
	return t.Flush()
}

func printTagDetails(out io.Writer, tag *models.PlanTag) error {
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)

	fmt.Fprintf(tw, "Name:\t%s\n", tag.Name)
	if tag.Spec != nil {
		fmt.Fprintf(tw, "Plan ID:\t%s\n", tag.Spec.PlanID)
	}
	if tag.Plan != nil && tag.Plan.Spec != nil && tag.Plan.Spec.SelectionHint != "" {
		fmt.Fprintf(tw, "Selection Hint:\t%s\n", tag.Plan.Spec.SelectionHint)
	}
	if tag.Plan != nil && tag.Plan.Status != nil {
		if by := planshared.FormatCreatedBy(tag.Plan.Status.CreatedBy); by != "" {
			// "Plan Created By" rather than "Created By" so it's
			// unambiguous against the tag-level Created/Updated fields
			// just below.
			fmt.Fprintf(tw, "Plan Created By:\t%s\n", by)
		}
	}
	if tag.Status != nil {
		fmt.Fprintf(tw, "Created:\t%s\n", utils.FormatTimestamp(tag.Status.CreatedAt))
		fmt.Fprintf(tw, "Updated:\t%s\n", utils.FormatTimestamp(tag.Status.UpdatedAt))
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	// If the tag has an inlined plan, render its body (Inputs / Steps
	// / Outputs) with the same shape plan get uses. Plan-level
	// metadata (ID, prompt, runner, ...) is omitted: ID and
	// SelectionHint are already at the tag level above; the rest is
	// one signadot plan get away.
	if tag.Plan != nil {
		planshared.PrintPlanBody(out, tag.Plan)
	}

	// Print tag history if present.
	if len(tag.History) > 1 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "History:")
		return printHistoryTable(out, tag.History)
	}

	return nil
}

type historyRow struct {
	PlanID    string `sdtab:"PLAN ID"`
	TaggedAt  string `sdtab:"TAGGED"`
	UntaggedAt string `sdtab:"UNTAGGED"`
}

func printHistoryTable(out io.Writer, history []*models.TagMapping) error {
	t := sdtab.New[historyRow](out)
	t.AddHeader()
	for _, h := range history {
		untagged := "(current)"
		if h.UntaggedAt != "" {
			untagged = utils.TimeAgo(h.UntaggedAt)
		}
		t.AddRow(historyRow{
			PlanID:     h.PlanID,
			TaggedAt:   utils.TimeAgo(h.TaggedAt),
			UntaggedAt: untagged,
		})
	}
	return t.Flush()
}

