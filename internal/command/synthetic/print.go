package synthetic

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
)

func printTest(oFmt config.OutputFormat, w io.Writer, t *models.Test) error {
	switch oFmt {
	case config.OutputFormatDefault:
		return printTestDetails(w, t)
	case config.OutputFormatJSON:
		return print.RawJSON(w, t)
	case config.OutputFormatYAML:
		return print.RawYAML(w, t)
	default:
		return fmt.Errorf("unsupported output format: %q", oFmt)
	}
}

func printTestDetails(out io.Writer, t *models.Test) error {
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)

	fmt.Fprintf(tw, "Name:\t%s\n", t.Name)
	fmt.Fprintf(tw, "Created:\t%s\n", utils.FormatTimestamp(t.CreatedAt))
	fmt.Fprintf(tw, "Updated:\t%s\n", utils.FormatTimestamp(t.UpdatedAt))
	return tw.Flush()
}
