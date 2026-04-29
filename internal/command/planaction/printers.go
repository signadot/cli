package planaction

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/glamour"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
	"golang.org/x/term"
)

type actionRow struct {
	Name        string `sdtab:"NAME"`
	ManagedBy   string `sdtab:"MANAGED BY"`
	Enabled     string `sdtab:"ENABLED"`
	Revision    string `sdtab:"REVISION"`
	Description string `sdtab:"DESCRIPTION"`
	Updated     string `sdtab:"UPDATED"`
}

func printActionTable(out io.Writer, actions []*models.PlanAction) error {
	t := sdtab.New[actionRow](out)
	t.AddHeader()
	for _, a := range actions {
		var revision, description, updated string
		if a.Status != nil {
			if a.Status.Revision != 0 {
				revision = fmt.Sprintf("%d", a.Status.Revision)
			}
			description = a.Status.Description
			updated = utils.FormatTimestamp(a.Status.UpdatedAt)
		}
		t.AddRow(actionRow{
			Name:        a.Name,
			ManagedBy:   a.ManagedBy,
			Enabled:     enabled(a),
			Revision:    revision,
			Description: description,
			Updated:     updated,
		})
	}
	return t.Flush()
}

func printActionDetails(out io.Writer, a *models.PlanAction) error {
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	fmt.Fprintf(tw, "Name:\t%s\n", a.Name)
	if a.ID != "" {
		fmt.Fprintf(tw, "ID:\t%s\n", a.ID)
	}
	if a.ManagedBy != "" {
		fmt.Fprintf(tw, "Managed by:\t%s\n", a.ManagedBy)
	}
	fmt.Fprintf(tw, "Enabled:\t%s\n", enabled(a))
	if a.Status != nil {
		if a.Status.Description != "" {
			fmt.Fprintf(tw, "Description:\t%s\n", a.Status.Description)
		}
		if a.Status.Revision != 0 {
			fmt.Fprintf(tw, "Revision:\t%d\n", a.Status.Revision)
		}
		if a.Status.CreatedAt != "" {
			fmt.Fprintf(tw, "Created:\t%s\n", utils.FormatTimestamp(a.Status.CreatedAt))
		}
		if a.Status.UpdatedAt != "" {
			fmt.Fprintf(tw, "Updated:\t%s\n", utils.FormatTimestamp(a.Status.UpdatedAt))
		}
		if len(a.Status.Requires) > 0 {
			fmt.Fprintf(tw, "Requires:\t%s\n", strings.Join(a.Status.Requires, ", "))
		}
		if img := formatImage(a.Status.BodyImage); img != "" {
			fmt.Fprintf(tw, "Image:\t%s\n", img)
		}
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	if a.Status != nil {
		if err := printFields(out, "Inputs", a.Status.BodyParams); err != nil {
			return err
		}
		if err := printFields(out, "Outputs", a.Status.BodyOutputs); err != nil {
			return err
		}
		if err := printValidations(out, a.Status.Validations); err != nil {
			return err
		}
	}

	if a.Spec != nil && a.Spec.Body != "" {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Body:")
		printBody(out, a.Spec.Body)
	}

	return nil
}

func printBody(out io.Writer, body string) {
	body = strings.TrimRight(body, "\n")
	rendered, ok := renderMarkdown(out, body)
	if !ok {
		fmt.Fprintln(out, body)
		return
	}
	fmt.Fprint(out, rendered)
}

func renderMarkdown(out io.Writer, body string) (string, bool) {
	f, ok := out.(*os.File)
	if !ok || !term.IsTerminal(int(f.Fd())) {
		return "", false
	}
	width, _, err := term.GetSize(int(f.Fd()))
	if err != nil || width <= 0 {
		width = 100
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", false
	}
	rendered, err := r.Render(body)
	if err != nil {
		return "", false
	}
	return rendered, true
}

func enabled(a *models.PlanAction) string {
	if a.Status == nil {
		return ""
	}
	return fmt.Sprintf("%t", a.Status.Enabled)
}

func formatImage(ref *models.PlanImageRef) string {
	if ref == nil {
		return ""
	}
	if ref.Literal != "" {
		return ref.Literal
	}
	if ref.InputName != "" {
		return fmt.Sprintf("input=%s", ref.InputName)
	}
	return ""
}

type fieldRow struct {
	Name     string `sdtab:"NAME"`
	Required string `sdtab:"REQUIRED"`
	Default  string `sdtab:"DEFAULT"`
	Schema   string `sdtab:"SCHEMA"`
}

func printFields(out io.Writer, label string, fields []*models.PlanField) error {
	if len(fields) == 0 {
		return nil
	}
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s:\n", label)
	t := sdtab.New[fieldRow](out)
	t.AddHeader()
	for _, f := range fields {
		t.AddRow(fieldRow{
			Name:     f.Name,
			Required: fmt.Sprintf("%t", f.Required),
			Default:  formatAny(f.Default),
			Schema:   formatSchema(f),
		})
	}
	return t.Flush()
}

type validationRow struct {
	RunnerGroup string `sdtab:"RUNNER GROUP"`
	Valid       string `sdtab:"VALID"`
	Stale       string `sdtab:"STALE"`
	Validated   string `sdtab:"VALIDATED"`
}

func printValidations(out io.Writer, vs []*models.PlanActionValidationStatus) error {
	if len(vs) == 0 {
		return nil
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Validations:")
	t := sdtab.New[validationRow](out)
	t.AddHeader()
	for _, v := range vs {
		t.AddRow(validationRow{
			RunnerGroup: v.RunnerGroup,
			Valid:       fmt.Sprintf("%t", v.Valid),
			Stale:       fmt.Sprintf("%t", v.Stale),
			Validated:   utils.FormatTimestamp(v.ValidatedAt),
		})
	}
	return t.Flush()
}

func formatAny(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func formatSchema(f *models.PlanField) string {
	if f.SchemaRef != "" {
		return f.SchemaRef
	}
	if f.Schema == nil {
		return ""
	}
	if m, ok := f.Schema.(map[string]any); ok {
		if t, ok := m["type"].(string); ok && t != "" {
			return t
		}
	}
	return "(custom)"
}
