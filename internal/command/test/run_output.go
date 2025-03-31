package test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/go-sdk/models"
	"github.com/theckman/yacspin"
)

type defaultRunOutput struct {
	sync.Mutex

	cfg   *config.TestRun
	wOut  io.Writer
	runID string
	txs   []*models.TestExecution
}

func newDefaultRunOutput(cfg *config.TestRun, wOut io.Writer, runID string) *defaultRunOutput {
	return &defaultRunOutput{
		cfg:   cfg,
		wOut:  wOut,
		runID: runID,
	}
}

func (o *defaultRunOutput) start() {
	fmt.Fprintf(o.wOut, "Created test run %q in cluster %q.\n\n", o.runID, o.cfg.Cluster)
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

	fmt.Fprintf(o.wOut, "Test run status:\n")
	for _, tx := range txs {
		var icon, statusText string
		switch tx.Status.Phase {
		case "pending":
			icon = runningIcon
			statusText = "pending"
		case "in_progress":
			icon = runningIcon
			statusText = "running"
		case "failed":
			icon = "❌"
			statusText = "failed"
		case "canceled":
			icon = "❌"
			statusText = "canceled"
		case "succeeded":
			icon = "✅"
			statusText = "completed"
		default:
			icon = "⚪"
			statusText = tx.Status.Phase
		}

		// add some padding to completely overwrite lines
		padding := "      "
		fmt.Fprintf(o.wOut, "%s\t%s\t[%s]%s\n", icon, tx.Spec.EmbeddedSpec.TestName, statusText, padding)
	}
}

func (o *defaultRunOutput) renderTestXsSummary(txs []*models.TestExecution) {
	fmt.Fprint(o.wOut, "\nTest run summary:\n")

	fmt.Fprint(o.wOut, "* Executions\n")
	fmt.Fprint(o.wOut, "\t"+o.getExecutionsDetails(txs)+"\n")

	diffMsg := o.getDiffsDetails(txs)
	if diffMsg != "" {
		fmt.Fprint(o.wOut, "* Diffs\n")
		fmt.Fprint(o.wOut, "\t"+diffMsg+"\n")
	}

	checksMsg := o.getChecksDetails(txs)
	if checksMsg != "" {
		fmt.Fprint(o.wOut, "* Checks\n")
		fmt.Fprint(o.wOut, "\t"+checksMsg+"\n")
	}
	fmt.Fprint(o.wOut, "\n")
}

func (o *defaultRunOutput) getExecutionsDetails(txs []*models.TestExecution) string {
	total := 0
	phaseMap := map[string]int{}
	for _, tx := range txs {
		total += 1
		phaseMap[tx.Status.Phase] += 1
	}

	var icon string
	switch {
	case phaseMap["canceled"] > 0 || phaseMap["failed"] > 0:
		icon = "❌"
	default:
		icon = "✅"
	}

	details := fmt.Sprintf("%s %d/%d tests completed", icon, phaseMap["succeeded"], total)
	if phaseMap["succeeded"] != total {
		details += " ("
		var otherSts []string
		if phaseMap["canceled"] > 0 {
			otherSts = append(otherSts,
				fmt.Sprintf("%d canceled", phaseMap["canceled"]))
		}
		if phaseMap["failed"] > 0 {
			otherSts = append(otherSts,
				fmt.Sprintf("%d failed", phaseMap["failed"]))
		}
		details += strings.Join(otherSts, ", ") + ")"
	}
	return details
}

func (o *defaultRunOutput) getDiffsDetails(txs []*models.TestExecution) string {
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

func (o *defaultRunOutput) getChecksDetails(txs []*models.TestExecution) string {
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

func structuredOutput(cfg *config.TestRun, outW io.Writer, runID string,
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
