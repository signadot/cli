package planexec

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/docker/go-units"
	"github.com/go-openapi/runtime"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/client"
	planlogs "github.com/signadot/go-sdk/client/plan_execution_logs"
	planexecs "github.com/signadot/go-sdk/client/plan_executions"
	"github.com/signadot/go-sdk/models"
	"github.com/signadot/go-sdk/transport"
	"github.com/spf13/cobra"
)

func newLogs(exec *config.PlanExecution) *cobra.Command {
	cfg := &config.PlanExecLogs{PlanExecution: exec}

	cmd := &cobra.Command{
		Use:   "logs EXECUTION_ID [STEP_ID]",
		Short: "List, download, or stream plan execution logs",
		Long: `List captured logs, download a captured log, or stream live logs.

List captured logs (table):
  signadot plan x logs <exec-id>

Download a captured step log:
  signadot plan x logs <exec-id> <step-id>              # stdout (default)
  signadot plan x logs <exec-id> <step-id> -s stderr    # stderr

Stream live logs (follow):
  signadot plan x logs <exec-id> -f                     # aggregated, all steps
  signadot plan x logs <exec-id> <step-id> -f           # single step

Bulk export captured logs:
  signadot plan x logs <exec-id> --all --dir ./logs/`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogs(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}

	cfg.AddFlags(cmd)
	return cmd
}

func runLogs(cfg *config.PlanExecLogs, out, log io.Writer, args []string) error {
	execID := args[0]
	stepID := ""
	if len(args) > 1 {
		stepID = args[1]
	}

	// Validate flag combinations.
	if cfg.All {
		if stepID != "" {
			return fmt.Errorf("--all does not accept a STEP_ID argument")
		}
		if cfg.Dir == "" {
			return fmt.Errorf("--all requires --dir")
		}
		if cfg.Follow {
			return fmt.Errorf("--all cannot be combined with -f")
		}
	}

	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	switch {
	case cfg.Follow:
		// Live streaming requires an active runner; warn if the execution
		// already terminated (the runner is gone, logs are captured instead).
		if terminal, phase, err := execTerminalPhase(cfg, execID); err != nil {
			return err
		} else if terminal {
			return fmt.Errorf("execution is %s; live logs are unavailable. Omit -f to read captured logs", phase)
		}
		ctx, cancel := signal.NotifyContext(context.Background(),
			os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
		defer cancel()
		return showPlanLogs(ctx, cfg.API, out, execID, stepID, cfg.Stream, int(cfg.TailLines))
	case cfg.All:
		return bulkExportLogs(cfg, log, execID)
	case stepID != "":
		return downloadStepLog(cfg, out, execID, stepID)
	default:
		return listCapturedLogs(cfg, out, execID)
	}
}

// listCapturedLogs prints a table of captured log streams.
func listCapturedLogs(cfg *config.PlanExecLogs, out io.Writer, execID string) error {
	params := planexecs.NewGetPlanExecutionParams().
		WithOrgName(cfg.Org).
		WithExecutionID(execID)
	resp, err := cfg.Client.PlanExecutions.GetPlanExecution(params, nil)
	if err != nil {
		return err
	}
	return renderCapturedLogs(out, resp.Payload, cfg.OutputFormat)
}

// renderCapturedLogs writes the captured-log view for an execution. Extracted
// from listCapturedLogs so it can be tested without an API round-trip.
func renderCapturedLogs(out io.Writer, ex *models.PlanExecution, format config.OutputFormat) error {
	logs := collectAllLogs(ex)
	running := ex != nil && ex.Status != nil && !isTerminalPhase(ex.Status.Phase)

	switch format {
	case config.OutputFormatDefault:
		if running {
			fmt.Fprintln(out, "Execution is still running. Captured logs appear as steps complete. Use -f to stream live.")
			fmt.Fprintln(out)
		}
		return printLogsTable(out, logs)
	case config.OutputFormatJSON:
		return print.RawJSON(out, logs)
	case config.OutputFormatYAML:
		return print.RawYAML(out, logs)
	default:
		return fmt.Errorf("unsupported output format: %q", format)
	}
}

// downloadStepLog downloads a single captured step log to out.
func downloadStepLog(cfg *config.PlanExecLogs, out io.Writer, execID, stepID string) error {
	transportCfg := cfg.GetBaseTransport()
	transportCfg.OverrideConsumers = true
	transportCfg.Consumers = map[string]runtime.Consumer{
		"*/*": runtime.ByteStreamConsumer(),
	}

	return cfg.APIClientWithCustomTransport(transportCfg,
		func(c *client.SignadotAPI) error {
			params := planlogs.NewDownloadStepLogParams().
				WithTimeout(4*time.Minute).
				WithOrgName(cfg.Org).
				WithExecutionID(execID).
				WithStepID(stepID).
				WithStream(cfg.Stream)
			_, _, err := c.PlanExecutionLogs.DownloadStepLog(params, nil, out)
			if err == nil {
				return nil
			}

			// On 404, try to give a more helpful message.
			var apiErr *transport.APIError
			if errors.As(err, &apiErr) && apiErr.Code == 404 {
				if hint := diagnoseMissingLog(cfg, execID, stepID); hint != "" {
					return errors.New(hint)
				}
			}
			return fmt.Errorf("downloading log for step %q: %w", stepID, err)
		})
}

// diagnoseMissingLog fetches the execution to explain why a log wasn't found.
func diagnoseMissingLog(cfg *config.PlanExecLogs, execID, stepID string) string {
	params := planexecs.NewGetPlanExecutionParams().
		WithOrgName(cfg.Org).
		WithExecutionID(execID)
	resp, err := cfg.Client.PlanExecutions.GetPlanExecution(params, nil)
	if err != nil || resp.Payload.Status == nil {
		return ""
	}
	var step *models.PlanStepStatus
	for _, s := range resp.Payload.Status.Steps {
		if s.ID == stepID {
			step = s
			break
		}
	}
	if step == nil {
		return fmt.Sprintf("step %q not found in execution", stepID)
	}
	if !isTerminalStepPhase(step.Phase) {
		return fmt.Sprintf("step %q is still %s; use -f to stream live logs", stepID, step.Phase)
	}
	if len(step.Logs) == 0 {
		return fmt.Sprintf("step %q has no captured logs", stepID)
	}
	var available []string
	for _, l := range step.Logs {
		available = append(available, string(l.Stream))
	}
	return fmt.Sprintf("step %q has no %s log (available: %s)", stepID, cfg.Stream, strings.Join(available, ", "))
}

// bulkExportLogs downloads all captured logs to a directory layout of <dir>/<stepID>/<stream>.
func bulkExportLogs(cfg *config.PlanExecLogs, log io.Writer, execID string) error {
	getParams := planexecs.NewGetPlanExecutionParams().
		WithOrgName(cfg.Org).
		WithExecutionID(execID)
	resp, err := cfg.Client.PlanExecutions.GetPlanExecution(getParams, nil)
	if err != nil {
		return err
	}

	if resp.Payload.Status == nil || len(resp.Payload.Status.Steps) == 0 {
		fmt.Fprintln(log, "No steps with logs.")
		return nil
	}

	type stepLog struct {
		StepID string
		Stream string
	}
	var logs []stepLog
	for _, s := range resp.Payload.Status.Steps {
		for _, l := range s.Logs {
			logs = append(logs, stepLog{StepID: s.ID, Stream: string(l.Stream)})
		}
	}
	if len(logs) == 0 {
		fmt.Fprintln(log, "No logs captured yet.")
		return nil
	}

	if err := os.MkdirAll(cfg.Dir, 0o755); err != nil {
		return err
	}

	transportCfg := cfg.GetBaseTransport()
	transportCfg.OverrideConsumers = true
	transportCfg.Consumers = map[string]runtime.Consumer{
		"*/*": runtime.ByteStreamConsumer(),
	}

	return cfg.APIClientWithCustomTransport(transportCfg,
		func(c *client.SignadotAPI) error {
			for _, sl := range logs {
				stepDir := filepath.Join(cfg.Dir, sl.StepID)
				if err := os.MkdirAll(stepDir, 0o755); err != nil {
					return fmt.Errorf("creating %s: %w", stepDir, err)
				}
				outPath := filepath.Join(stepDir, sl.Stream)

				f, err := os.Create(outPath)
				if err != nil {
					return fmt.Errorf("creating %s: %w", outPath, err)
				}

				params := planlogs.NewDownloadStepLogParams().
					WithTimeout(4*time.Minute).
					WithOrgName(cfg.Org).
					WithExecutionID(execID).
					WithStepID(sl.StepID).
					WithStream(sl.Stream)
				_, _, dlErr := c.PlanExecutionLogs.DownloadStepLog(params, nil, f)
				f.Close()
				if dlErr != nil {
					return fmt.Errorf("downloading %s/%s: %w", sl.StepID, sl.Stream, dlErr)
				}
				fmt.Fprintf(log, "Exported %s\n", outPath)
			}
			return nil
		})
}

// --- Live streaming (follow) ---

func showPlanLogs(ctx context.Context, cfg *config.API, out io.Writer, execID, stepID, stream string, tailLines int) error {
	transportCfg := cfg.GetBaseTransport()
	transportCfg.Consumers = map[string]runtime.Consumer{
		"text/event-stream": runtime.ByteStreamConsumer(),
	}

	return cfg.APIClientWithCustomTransport(transportCfg,
		func(c *client.SignadotAPI) error {
			reader, writer := io.Pipe()

			errch := make(chan error, 2)

			go func() {
				_, err := print.ParseSSEStream(reader, out)
				if errors.Is(err, io.ErrClosedPipe) {
					err = nil
				}
				reader.Close()
				errch <- err
			}()

			go func() {
				var err error
				if stepID != "" {
					params := planlogs.NewStreamPlanExecutionStepLogsParams().
						WithContext(ctx).
						WithTimeout(0).
						WithOrgName(cfg.Org).
						WithExecutionID(execID).
						WithStepID(stepID).
						WithStream(stream)
					if tailLines > 0 {
						tl := int64(tailLines)
						params.WithTailLines(&tl)
					}
					_, err = c.PlanExecutionLogs.StreamPlanExecutionStepLogs(params, nil, writer)
				} else {
					params := planlogs.NewStreamPlanExecutionLogsParams().
						WithContext(ctx).
						WithTimeout(0).
						WithOrgName(cfg.Org).
						WithExecutionID(execID)
					if tailLines > 0 {
						tl := int64(tailLines)
						params.WithTailLines(&tl)
					}
					_, err = c.PlanExecutionLogs.StreamPlanExecutionLogs(params, nil, writer)
				}
				if errors.Is(err, io.ErrClosedPipe) {
					err = nil
				}
				writer.Close()
				errch <- err
			}()

			return errors.Join(<-errch, <-errch)
		})
}

// --- Captured log listing ---

type logEntry struct {
	Step    string `json:"step"`
	Stream  string `json:"stream"`
	Storage string `json:"storage"`
	Size    int64  `json:"size,omitempty"`
	Ready   *bool  `json:"ready,omitempty"`
}

func collectAllLogs(ex *models.PlanExecution) []logEntry {
	if ex.Status == nil {
		return nil
	}
	var entries []logEntry
	for _, s := range ex.Status.Steps {
		for _, l := range s.Logs {
			e := logEntry{
				Step:   s.ID,
				Stream: string(l.Stream),
			}
			if l.Artifact != nil {
				e.Storage = "artifact"
				e.Size = l.Artifact.Size
				e.Ready = &l.Artifact.Ready
			} else {
				e.Storage = "inline"
				e.Size = int64(len(l.Value))
				t := true
				e.Ready = &t
			}
			entries = append(entries, e)
		}
	}
	return entries
}

type logRow struct {
	Step    string `sdtab:"STEP"`
	Stream  string `sdtab:"STREAM"`
	Storage string `sdtab:"STORAGE"`
	Size    string `sdtab:"SIZE"`
	Ready   string `sdtab:"READY"`
}

func printLogsTable(out io.Writer, logs []logEntry) error {
	t := sdtab.New[logRow](out)
	t.AddHeader()
	for _, l := range logs {
		size := ""
		ready := "-"
		if l.Size > 0 {
			size = units.HumanSize(float64(l.Size))
		}
		if l.Ready != nil {
			if *l.Ready {
				ready = "true"
			} else {
				ready = "false"
			}
		}
		t.AddRow(logRow{
			Step:    l.Step,
			Stream:  l.Stream,
			Storage: l.Storage,
			Size:    size,
			Ready:   ready,
		})
	}
	return t.Flush()
}

// --- Phase helpers ---

// execTerminalPhase fetches the execution and reports whether it has reached
// a terminal phase (completed, failed, cancelled).
func execTerminalPhase(cfg *config.PlanExecLogs, execID string) (bool, models.PlansExecutionPhase, error) {
	params := planexecs.NewGetPlanExecutionParams().
		WithOrgName(cfg.Org).
		WithExecutionID(execID)
	resp, err := cfg.Client.PlanExecutions.GetPlanExecution(params, nil)
	if err != nil {
		return false, "", err
	}
	if resp.Payload.Status == nil {
		return false, "", nil
	}
	return isTerminalPhase(resp.Payload.Status.Phase), resp.Payload.Status.Phase, nil
}

func isTerminalPhase(phase models.PlansExecutionPhase) bool {
	switch phase {
	case models.PlansExecutionPhaseCompleted,
		models.PlansExecutionPhaseFailed,
		models.PlansExecutionPhaseCancelled:
		return true
	}
	return false
}

func isTerminalStepPhase(phase models.PlansStepPhase) bool {
	switch phase {
	case models.PlansStepPhaseCompleted,
		models.PlansStepPhaseFailed,
		models.PlansStepPhaseSkipped:
		return true
	}
	return false
}
