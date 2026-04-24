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
	var out, log bytes.Buffer
	if err := renderCapturedLogs(&out, &log, ex, config.OutputFormatDefault); err != nil {
		t.Fatalf("renderCapturedLogs: %v", err)
	}
	if !strings.Contains(log.String(), "Execution is still running") {
		t.Errorf("expected running hint on stderr, got:\n%s", log.String())
	}
	if strings.Contains(out.String(), "Execution is still running") {
		t.Errorf("running hint should not appear on stdout:\n%s", out.String())
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
			var out, log bytes.Buffer
			if err := renderCapturedLogs(&out, &log, ex, config.OutputFormatDefault); err != nil {
				t.Fatalf("renderCapturedLogs: %v", err)
			}
			if strings.Contains(log.String(), "Execution is still running") {
				t.Errorf("unexpected running hint for phase %s:\n%s", phase, log.String())
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
	var out, log bytes.Buffer
	if err := renderCapturedLogs(&out, &log, ex, config.OutputFormatJSON); err != nil {
		t.Fatalf("renderCapturedLogs: %v", err)
	}
	// JSON output must not carry the human-facing hint.
	if strings.Contains(out.String(), "Execution is still running") {
		t.Errorf("running hint leaked into JSON output:\n%s", out.String())
	}
	// But the log entry should still be present.
	if !strings.Contains(out.String(), `"step": "step1"`) {
		t.Errorf("expected step1 entry in JSON output, got:\n%s", out.String())
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
	var out, log bytes.Buffer
	if err := renderCapturedLogs(&out, &log, ex, config.OutputFormatDefault); err != nil {
		t.Fatalf("renderCapturedLogs: %v", err)
	}
	if !strings.Contains(log.String(), "Execution is still running") {
		t.Errorf("expected running hint on stderr, got:\n%s", log.String())
	}
	if !strings.Contains(out.String(), "done_step") || !strings.Contains(out.String(), "stderr") {
		t.Errorf("expected done_step stderr row in table, got:\n%s", out.String())
	}
	if strings.Contains(out.String(), "running_step") {
		t.Errorf("running_step has no logs yet and should not appear, got:\n%s", out.String())
	}
}

func TestRenderCapturedLogs_NilExecution(t *testing.T) {
	var out, log bytes.Buffer
	if err := renderCapturedLogs(&out, &log, nil, config.OutputFormatDefault); err != nil {
		t.Fatalf("renderCapturedLogs: %v", err)
	}
	// Should produce an empty table, no panic.
	if strings.Contains(out.String(), "Execution is still running") {
		t.Errorf("unexpected hint for nil execution:\n%s", out.String())
	}
}

func TestValidatePathComponent(t *testing.T) {
	for _, name := range []string{"step1", "check_response", "my-step"} {
		if err := validatePathComponent(name); err != nil {
			t.Errorf("validatePathComponent(%q) = %v; want nil", name, err)
		}
	}
	for _, name := range []string{"..", "../etc", "a/b", "/abs", `a\b`} {
		if err := validatePathComponent(name); err == nil {
			t.Errorf("validatePathComponent(%q) = nil; want error", name)
		}
	}
}
