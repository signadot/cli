package smarttest

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/go-sdk/models"
	"github.com/theckman/yacspin"
)

type defaultRunOutput struct {
	sync.Mutex

	cfg   *config.SmartTestRun
	wOut  io.Writer
	runID string
	txs   []*models.TestExecution
}

func newDefaultRunOutput(cfg *config.SmartTestRun, wOut io.Writer, runID string) *defaultRunOutput {
	return &defaultRunOutput{
		cfg:   cfg,
		wOut:  wOut,
		runID: runID,
	}
}

func (o *defaultRunOutput) start() {
	fmt.Fprintf(o.wOut, "Created test run with ID %q in cluster %q.\n\n", o.runID, o.cfg.Cluster)
}

func (o *defaultRunOutput) setTestXs(txs []*models.TestExecution) {
	o.Lock()
	defer o.Unlock()
	o.txs = txs
}

func (o *defaultRunOutput) updateTestXTable(ctx context.Context) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	charSet := yacspin.CharSets[14]
	charSetIndex := 0

	for {
		// define the running chart
		char := charSet[charSetIndex]
		charSetIndex = (charSetIndex + 1) % len(charSet)

		// update the output
		o.Lock()
		if len(o.txs) == 0 {
			fmt.Fprint(o.wOut, char+"\n")
			// move up the cursor 1 line
			fmt.Fprint(o.wOut, "\033[1A")
		} else {
			o.renderTestXsTable(o.txs, char)
			// Move the cursor up len(txs)+1 lines
			fmt.Fprintf(o.wOut, "\033[%dA", len(o.txs)+1)
		}
		o.Unlock()

		// wait until next update
		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return
		}
	}
}

func (o *defaultRunOutput) renderTestXsTable(txs []*models.TestExecution, runningIcon string) {
	if runningIcon == "" {
		runningIcon = "🟡"
	}

	tw := tabwriter.NewWriter(o.wOut, 0, 0, 3, ' ', 0)
	fmt.Fprintf(tw, "Test run status:\n")
	for _, tx := range txs {
		var icon, statusText string
		switch tx.Status.Phase {
		case models.TestexecutionsPhasePending:
			icon = runningIcon
			statusText = "pending"
		case models.TestexecutionsPhaseInProgress:
			icon = runningIcon
			statusText = "running"
		case models.TestexecutionsPhaseFailed:
			icon = "❌"
			statusText = "failed"
		case models.TestexecutionsPhaseCanceled:
			icon = "❌"
			statusText = "canceled"
		case models.TestexecutionsPhaseSucceeded:
			icon = "✅"
			statusText = "completed"
		default:
			icon = "⚪"
			statusText = string(tx.Status.Phase)
		}
		fmt.Fprintf(tw, "%s\t%s\t[ID: %s, STATUS: %s]\n", icon, truncateTestName(tx.Spec.External.TestName, 48),
			tx.ID, statusText)
	}
	tw.Flush()
}

func (o *defaultRunOutput) renderTestXsSummary(txs []*models.TestExecution) {
	tw := tabwriter.NewWriter(o.wOut, 0, 0, 3, ' ', 0)
	fmt.Fprint(tw, "\nTest run summary:\n")

	fmt.Fprint(tw, "* Executions\n")
	fmt.Fprint(tw, "\t"+o.getExecutionsDetails(txs)+"\n")

	diffMsg := getDiffsDetails(txs...)
	if diffMsg != "" {
		fmt.Fprint(tw, "* Diffs\n")
		fmt.Fprint(tw, "\t"+diffMsg+"\n")
	}

	checksMsg := getChecksDetails(txs...)
	if checksMsg != "" {
		fmt.Fprint(tw, "* Checks\n")
		fmt.Fprint(tw, "\t"+checksMsg+"\n")
	}
	fmt.Fprint(tw, "\n")
	tw.Flush()
}

func (o *defaultRunOutput) getExecutionsDetails(txs []*models.TestExecution) string {
	total := 0
	phaseMap := map[models.TestexecutionsPhase]int{}
	for _, tx := range txs {
		total += 1
		phaseMap[tx.Status.Phase] += 1
	}

	var icon string
	switch {
	case phaseMap[models.TestexecutionsPhaseCanceled] > 0 || phaseMap[models.TestexecutionsPhaseFailed] > 0:
		icon = "❌"
	default:
		icon = "✅"
	}

	details := fmt.Sprintf("%s %d/%d tests completed", icon, phaseMap[models.TestexecutionsPhaseSucceeded], total)
	if phaseMap[models.TestexecutionsPhaseSucceeded] != total {
		details += " ("
		var otherSts []string
		if phaseMap[models.TestexecutionsPhaseCanceled] > 0 {
			otherSts = append(otherSts,
				fmt.Sprintf("%d canceled", phaseMap[models.TestexecutionsPhaseCanceled]))
		}
		if phaseMap[models.TestexecutionsPhaseFailed] > 0 {
			otherSts = append(otherSts,
				fmt.Sprintf("%d failed", phaseMap[models.TestexecutionsPhaseFailed]))
		}
		details += strings.Join(otherSts, ", ") + ")"
	}
	return details
}

func getDiffsDetails(txs ...*models.TestExecution) string {
	var caps, red, yellow, green int64
	for _, tx := range txs {
		if tx.Results == nil || tx.Results.TrafficDiff == nil {
			continue
		}
		c, r, y, g := diffCounts(tx.Results.TrafficDiff)
		caps += c
		red += r
		yellow += y
		green += g
	}

	if caps == 0 {
		return ""
	}

	bold := color.New(color.FgHiWhite, color.Bold)

	switch {
	case red > 0 && yellow > 0:
		return fmt.Sprintf("⚠️ %s and %s relevance differences found",
			bold.Sprintf("%d high", red), bold.Sprintf("%d medium", yellow))
	case red > 0 && yellow == 0:
		return fmt.Sprintf("⚠️ %s relevance differences found",
			bold.Sprintf("%d high", red))
	case red == 0 && yellow > 0:
		return fmt.Sprintf("⚠️ %s relevance differences found",
			bold.Sprintf("%d medium", yellow))
	default:
		return "✅ No " + bold.Sprint("high") + "/" + bold.Sprint("medium") + " relevance differences found"
	}
}

func getChecksDetails(txs ...*models.TestExecution) string {
	var passed, failed int
	for _, tx := range txs {
		if tx.Results == nil || tx.Results.Checks == nil {
			continue
		}
		p, f := checksPassedFailed(tx.Results.Checks)
		passed += p
		failed += f
	}
	if passed+failed == 0 {
		return ""
	}

	if failed > 0 {
		return fmt.Sprintf("❌ %d checks passed, %d failed", passed, failed)
	}
	return fmt.Sprintf("✅ %d checks passed", passed)
}

func diffCounts(diff *models.TrafficDiff) (int64, int64, int64, int64) {
	if diff == nil {
		return 0, 0, 0, 0
	}
	var (
		red, green, yellow int64
	)
	if diff.Red != nil {
		red += diff.Red.Additions
		red += diff.Red.Removals
		red += diff.Red.Replacements
	}
	if diff.Yellow != nil {
		yellow += diff.Yellow.Additions
		yellow += diff.Yellow.Removals
		yellow += diff.Yellow.Replacements
	}
	if diff.Green != nil {
		green += diff.Green.Additions
		green += diff.Green.Removals
		green += diff.Green.Replacements
	}
	return diff.Captures, red, yellow, green
}

func checksPassedFailed(cks *models.TestexecutionsChecks) (int, int) {
	if cks == nil {
		return 0, 0
	}
	var (
		passed, failed int
	)
	for _, ck := range cks.Sandbox {
		if len(ck.Errors) == 0 {
			passed++
			continue
		}
		failed++
	}
	return passed, failed
}

func structuredOutput(cfg *config.SmartTestRun, outW io.Writer, runID string,
	txs []*models.TestExecution) error {
	type output struct {
		RunID      string `json:"runID"`
		Executions []*models.TestExecution
	}

	o := output{
		RunID:      runID,
		Executions: txs,
	}

	switch cfg.OutputFormat {
	case config.OutputFormatJSON:
		return print.RawJSON(outW, o)
	case config.OutputFormatYAML:
		return print.RawYAML(outW, o)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
