package smarttest

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/sdtab"
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

	if tx.Spec.EmbeddedSpec != nil {
		spec := tx.Spec.EmbeddedSpec
		// this is an external test
		fmt.Fprint(tw, "Source:\texternal\n")
		fmt.Fprintf(tw, "RunID:\t%s\n", spec.RunID)
		fmt.Fprintf(tw, "TestName:\t%s\n", spec.TestName)
		if spec.Repo != "" {
			// this is a git test
			fmt.Fprintf(tw, "Repo:\t%s\n", spec.Repo)
			fmt.Fprintf(tw, "Path:\t%s\n", spec.Path)
			fmt.Fprintf(tw, "Branch:\t%s\n", spec.Branch)
			fmt.Fprintf(tw, "CommitSHA:\t%s\n", spec.CommitSHA)
		}
		if len(spec.Labels) > 0 {
			fmt.Fprintf(tw, "Labels:\t%s\n", getLabels(spec.Labels))
		}
	} else {
		// this is a hosted test
		fmt.Fprint(tw, "Source:\thosted\n")
		fmt.Fprintf(tw, "TestName:\t%s\n", tx.Spec.Test)
		if tx.Status.TriggeredBy != nil && tx.Status.TriggeredBy.Sandbox != "" {
			fmt.Fprintf(tw, "TriggeredBy:\t%s\n", tx.Status.TriggeredBy.Sandbox)
		} else {
			fmt.Fprintf(tw, "TriggeredBy:\t%s\n", "-")
		}
	}

	if tx.Spec.ExecutionContext != nil {
		ec := tx.Spec.ExecutionContext
		fmt.Fprintf(tw, "Cluster:\t%s\n", ec.Cluster)
		if ec.Routing != nil {
			if ec.Routing.Sandbox != "" {
				fmt.Fprintf(tw, "Sandbox:\t%s\n", ec.Routing.Sandbox)
			} else if ec.Routing.Routegroup != "" {
				fmt.Fprintf(tw, "Routegroup:\t%s\n", ec.Routing.Routegroup)
			}
		}
	}

	fmt.Fprintf(tw, "Created:\t%s\n", utils.FormatTimestamp(tx.CreatedAt))
	fmt.Fprintf(tw, "Phase:\t%s", tx.Status.Phase)
	if tx.Status.FinalState != nil {
		if tx.Status.FinalState.Failed != nil {
			fmt.Fprint(tw, " ("+tx.Status.FinalState.Failed.Message+")")
		} else if tx.Status.FinalState.Canceled != nil {
			fmt.Fprint(tw, " ("+tx.Status.FinalState.Canceled.Message+")")
		}
	}
	fmt.Fprint(tw, "\n")

	resultsMsg := getResults(tx)
	if resultsMsg != "" {
		fmt.Fprintf(tw, "Results:\n%s", resultsMsg)
	}
	return tw.Flush()
}

func getLabels(labels map[string]string) string {
	labelStrs := make([]string, 0, len(labels))
	for k, v := range labels {
		labelStrs = append(labelStrs, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(labelStrs, ", ")
}

func getResults(tx *models.TestExecution) string {
	if tx.Results == nil {
		return ""
	}

	var resultsMsg string
	diffMsg := getDiffsDetails(tx)
	if diffMsg != "" {
		resultsMsg += "* Diffs\n"
		resultsMsg += fmt.Sprintf("\t" + diffMsg + "\n")
	}
	checksMsg := getChecksDetails(tx)
	if checksMsg != "" {
		resultsMsg += "* Checks\n"
		resultsMsg += fmt.Sprintf("\t" + checksMsg + "\n")
	}
	return resultsMsg
}

type testExecRow struct {
	Name      string `sdtab:"NAME"`
	Source    string `sdtab:"SOURCE"`
	Phase     string `sdtab:"PHASE"`
	CreatedAt string `sdtab:"CREATED"`
}

func printTestExecutionsTable(w io.Writer, txs []*models.TestexecutionsQueryResult) error {
	tab := sdtab.New[testExecRow](w)
	tab.AddHeader()
	for _, item := range txs {
		tx := item.Execution
		source := "hosted"
		if tx.Spec != nil && tx.Spec.EmbeddedSpec != nil {
			source = "external"
		}
		tab.AddRow(testExecRow{
			Name:      tx.Name,
			Source:    source,
			CreatedAt: tx.CreatedAt,
			Phase:     tx.Status.Phase,
		})
	}
	return tab.Flush()
}
