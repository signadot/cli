package planexec

import (
	"bytes"
	"strings"
	"testing"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/models"
)

func TestRenderCapturedLogs_RunningShowsHint(t *testing.T) {
	ex := &models.PlanExecution{
		Status: &models.PlanExecutionStatus{
			Phase: models.PlansExecutionPhaseRunning,
			Steps: []*models.PlanStepStatus{
				{ID: "step1", Phase: models.PlansStepPhaseRunning},
			},
		},
	}
	var buf bytes.Buffer
	if err := renderCapturedLogs(&buf, ex, config.OutputFormatDefault); err != nil {
		t.Fatalf("renderCapturedLogs: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Execution is still running") {
		t.Errorf("expected running hint, got:\n%s", out)
	}
}

func TestRenderCapturedLogs_TerminalNoHint(t *testing.T) {
	for _, phase := range []models.PlansExecutionPhase{
		models.PlansExecutionPhaseCompleted,
		models.PlansExecutionPhaseFailed,
		models.PlansExecutionPhaseCancelled,
	} {
		t.Run(string(phase), func(t *testing.T) {
			ex := &models.PlanExecution{
				Status: &models.PlanExecutionStatus{Phase: phase},
			}
			var buf bytes.Buffer
			if err := renderCapturedLogs(&buf, ex, config.OutputFormatDefault); err != nil {
				t.Fatalf("renderCapturedLogs: %v", err)
			}
			if strings.Contains(buf.String(), "Execution is still running") {
				t.Errorf("unexpected running hint for phase %s:\n%s", phase, buf.String())
			}
		})
	}
}

func TestRenderCapturedLogs_JSONSuppressesHint(t *testing.T) {
	ex := &models.PlanExecution{
		Status: &models.PlanExecutionStatus{
			Phase: models.PlansExecutionPhaseRunning,
			Steps: []*models.PlanStepStatus{
				{
					ID: "step1",
					Logs: []*models.PlanLogStatus{
						{Stream: models.LogsLogTypeStdout, Value: "hi"},
					},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := renderCapturedLogs(&buf, ex, config.OutputFormatJSON); err != nil {
		t.Fatalf("renderCapturedLogs: %v", err)
	}
	// JSON output must not carry the human-facing hint.
	if strings.Contains(buf.String(), "Execution is still running") {
		t.Errorf("running hint leaked into JSON output:\n%s", buf.String())
	}
	// But the log entry should still be present.
	if !strings.Contains(buf.String(), `"step": "step1"`) {
		t.Errorf("expected step1 entry in JSON output, got:\n%s", buf.String())
	}
}

func TestRenderCapturedLogs_RunningWithPartialLogs(t *testing.T) {
	ex := &models.PlanExecution{
		Status: &models.PlanExecutionStatus{
			Phase: models.PlansExecutionPhaseRunning,
			Steps: []*models.PlanStepStatus{
				// A step that already completed and captured stderr.
				{
					ID: "done_step",
					Logs: []*models.PlanLogStatus{
						{Stream: models.LogsLogTypeStderr, Value: "boom\n"},
					},
				},
				// A step that's still running with no captured logs yet.
				{ID: "running_step"},
			},
		},
	}
	var buf bytes.Buffer
	if err := renderCapturedLogs(&buf, ex, config.OutputFormatDefault); err != nil {
		t.Fatalf("renderCapturedLogs: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Execution is still running") {
		t.Errorf("expected running hint, got:\n%s", out)
	}
	if !strings.Contains(out, "done_step") || !strings.Contains(out, "stderr") {
		t.Errorf("expected done_step stderr row in table, got:\n%s", out)
	}
	if strings.Contains(out, "running_step") {
		t.Errorf("running_step has no logs yet and should not appear, got:\n%s", out)
	}
}
