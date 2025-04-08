package smarttest

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/models"
	"github.com/xeonx/timeago"
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
	fmt.Fprintf(tw, "ID:\t%s\n", tx.ID)

	if tx.Spec.External != nil {
		spec := tx.Spec.External
		// this is an external test
		fmt.Fprint(tw, "Source:\texternal\n")
		fmt.Fprintf(tw, "RunID:\t%s\n", tx.Spec.ExecutionContext.RunID)
		fmt.Fprintf(tw, "TestName:\t%s\n", spec.TestName)
		if spec.Repo != "" {
			// this is a git test
			fmt.Fprintf(tw, "Repo:\t%s\n", spec.Repo)
			fmt.Fprintf(tw, "Path:\t%s\n", spec.Path)
			fmt.Fprintf(tw, "Branch:\t%s\n", spec.Branch)
			fmt.Fprintf(tw, "CommitSHA:\t%s\n", spec.CommitSHA)
		}
	} else if tx.Spec.Hosted != nil {
		// this is a hosted test
		spec := tx.Spec.Hosted
		fmt.Fprint(tw, "Source:\thosted\n")
		fmt.Fprintf(tw, "TestName:\t%s\n", spec.TestName)
		if tx.Status.TriggeredBy != nil && tx.Status.TriggeredBy.Sandbox != "" {
			fmt.Fprintf(tw, "TriggeredBy:\t%s\n", tx.Status.TriggeredBy.Sandbox)
		} else {
			fmt.Fprintf(tw, "TriggeredBy:\t%s\n", "-")
		}
	} else {
		panic("invalid execution, neither hosted nor external")
	}
	if len(tx.Spec.Labels) > 0 {
		fmt.Fprintf(tw, "Labels:\t%s\n", getLabels(tx.Spec.Labels))
	}

	if tx.Spec.ExecutionContext != nil {
		ec := tx.Spec.ExecutionContext
		fmt.Fprintf(tw, "Cluster:\t%s\n", ec.Cluster)
		fmt.Fprintf(tw, "Environment:\t%s\n", getTXEnvironment(tx))
	}

	createdAt, duration := getTXCreatedAtAndDuration(tx)
	fmt.Fprintf(tw, "Created:\t%s\n", createdAt)
	if len(duration) != 0 {
		fmt.Fprintf(tw, "Duration:\t%s\n", duration)
	}
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
	ID          string `sdtab:"ID"`
	Source      string `sdtab:"SOURCE"`
	TestName    string `sdtab:"TESTNAME"`
	Environment string `sdtab:"ENVIRONMENT"`
	CreatedAt   string `sdtab:"CREATED AT"`
	Duration    string `sdtab:"DURATION"`
	Status      string `sdtab:"STATUS"`
}

func printTestExecutionsTable(w io.Writer, txs []*models.TestexecutionsQueryResult) error {
	tab := sdtab.New[testExecRow](w)
	tab.AddHeader()
	for _, item := range txs {
		tx := item.Execution
		source := "hosted"
		if tx.Spec != nil && tx.Spec.External != nil {
			source = "external"
		}
		var testName string
		if tx.Spec != nil {
			if tx.Spec.External != nil {
				testName = tx.Spec.External.TestName
			} else if tx.Spec.Hosted != nil {
				testName = tx.Spec.Hosted.TestName
			}
		}

		createdAt, duration := getTXCreatedAtAndDuration(tx)
		environment := getTXEnvironment(tx)

		tab.AddRow(testExecRow{
			ID:          tx.ID,
			Source:      source,
			TestName:    truncateTestName(testName, 48),
			Environment: environment,
			CreatedAt:   createdAt,
			Duration:    duration,
			Status:      tx.Status.Phase,
		})
	}
	return tab.Flush()
}

func truncateTestName(tn string, N int) string {
	if len(tn) > N && N > 3 {
		start := len(tn) - (N - 3)
		return "..." + tn[start:]
	}
	return tn
}

func getTXCreatedAtAndDuration(tx *models.TestExecution) (createdAtStr string, durationStr string) {
	var createdAt *time.Time

	createdAtRaw := tx.CreatedAt
	if len(createdAtRaw) != 0 {
		t, err := time.Parse(time.RFC3339, createdAtRaw)
		if err != nil {
			return "", ""
		}

		createdAt = &t
		createdAtStr = timeago.NoMax(timeago.English).Format(t)
	}

	finishedAtRaw := tx.Status.FinishedAt
	if createdAt != nil && len(finishedAtRaw) != 0 {
		finishedAt, err := time.Parse(time.RFC3339, finishedAtRaw)
		if err != nil {
			return "", ""
		}

		durationTime := finishedAt.Sub(*createdAt)
		durationStr = durationTime.String()
	}

	return createdAtStr, durationStr
}

func getTXEnvironment(tx *models.TestExecution) string {
	routingContext := tx.Spec.ExecutionContext.Routing

	switch {
	case routingContext == nil:
	case len(routingContext.Sandbox) > 0:
		return fmt.Sprintf("sandbox=%s", routingContext.Sandbox)
	case len(routingContext.Routegroup) > 0:
		return fmt.Sprintf("routegroup=%s", routingContext.Routegroup)
	}
	return "baseline"
}
