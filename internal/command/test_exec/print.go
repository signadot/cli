package test_exec

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
)

func PrintTestExecution(oFmt config.OutputFormat, w io.Writer, tx *models.TestExecution) error {
	switch oFmt {
	case config.OutputFormatDefault:
		return printTestExecutionDetails(w, tx)
	case config.OutputFormatJSON:
		return print.RawJSON(w, tx)
	case config.OutputFormatYAML:
		return print.RawYAML(w, tx)
	}
	return nil
}

func printTestExecutionDetails(w io.Writer, tx *models.TestExecution) error {
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)

	fmt.Fprintf(tw, "Name:\t%s\n", tx.Name)
	fmt.Fprintf(tw, "Test:\t%s\n", tx.Spec.Test)
	if tx.Status.TriggeredBy != nil && tx.Status.TriggeredBy.Sandbox != "" {
		fmt.Fprintf(tw, "TriggeredBy:\t%s\n", tx.Status.TriggeredBy.Sandbox)
	} else {
		fmt.Fprintf(tw, "TriggeredBy:\t%s\n", "-")
	}
	fmt.Fprintf(tw, "Created:\t%s\n", utils.FormatTimestamp(tx.CreatedAt))
	fmt.Fprintf(tw, "Phase:\t%s\n", tx.Status.Phase)
	return tw.Flush()
}
