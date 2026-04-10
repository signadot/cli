package plantag

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
	"github.com/xeonx/timeago"
)

type tagRow struct {
	Name    string `sdtab:"NAME"`
	PlanID  string `sdtab:"PLAN ID"`
	Steps   string `sdtab:"STEPS"`
	Prompt  string `sdtab:"PROMPT"`
	Created string `sdtab:"CREATED"`
	Updated string `sdtab:"UPDATED"`
}

func printTagTable(out io.Writer, tags []*models.PlanTag) error {
	t := sdtab.New[tagRow](out)
	t.AddHeader()
	for _, tag := range tags {
		var planID, steps, prompt, created, updated string
		if tag.Spec != nil {
			planID = tag.Spec.PlanID
		}
		if tag.Plan != nil && tag.Plan.Spec != nil {
			steps = fmt.Sprintf("%d", len(tag.Plan.Spec.Steps))
			prompt = print.FirstLine(tag.Plan.Spec.Prompt)
		}
		if tag.Status != nil {
			if tag.Status.CreatedAt != "" {
				if ts, err := time.Parse(time.RFC3339, tag.Status.CreatedAt); err == nil {
					created = timeago.NoMax(timeago.English).Format(ts)
				}
			}
			if tag.Status.UpdatedAt != "" {
				if ts, err := time.Parse(time.RFC3339, tag.Status.UpdatedAt); err == nil {
					updated = timeago.NoMax(timeago.English).Format(ts)
				}
			}
		}
		t.AddRow(tagRow{
			Name:    tag.Name,
			PlanID:  planID,
			Steps:   steps,
			Prompt:  prompt,
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
	if tag.Status != nil {
		fmt.Fprintf(tw, "Created:\t%s\n", utils.FormatTimestamp(tag.Status.CreatedAt))
		fmt.Fprintf(tw, "Updated:\t%s\n", utils.FormatTimestamp(tag.Status.UpdatedAt))
	}

	if err := tw.Flush(); err != nil {
		return err
	}

	// If the tag has an inlined plan, show its details.
	if tag.Plan != nil {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Plan:")
		tw = tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
		fmt.Fprintf(tw, "  ID:\t%s\n", tag.Plan.ID)
		if tag.Plan.Spec != nil {
			fmt.Fprintf(tw, "  Steps:\t%d\n", len(tag.Plan.Spec.Steps))
			if tag.Plan.Spec.Prompt != "" {
				fmt.Fprintf(tw, "  Prompt:\t%s\n", print.FirstLine(tag.Plan.Spec.Prompt))
			}
		}
		if tag.Plan.Status != nil {
			fmt.Fprintf(tw, "  Created:\t%s\n", utils.FormatTimestamp(tag.Plan.Status.CreatedAt))
		}
		if err := tw.Flush(); err != nil {
			return err
		}
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
		tagged := ""
		if h.TaggedAt != "" {
			if ts, err := time.Parse(time.RFC3339, h.TaggedAt); err == nil {
				tagged = timeago.NoMax(timeago.English).Format(ts)
			}
		}
		untagged := "(current)"
		if h.UntaggedAt != "" {
			if ts, err := time.Parse(time.RFC3339, h.UntaggedAt); err == nil {
				untagged = timeago.NoMax(timeago.English).Format(ts)
			}
		}
		t.AddRow(historyRow{
			PlanID:    h.PlanID,
			TaggedAt:  tagged,
			UntaggedAt: untagged,
		})
	}
	return t.Flush()
}

