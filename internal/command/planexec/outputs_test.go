package planexec

import (
	"fmt"
	"testing"

	"github.com/signadot/go-sdk/models"
)

func TestCollectAllOutputsDedup(t *testing.T) {
	ex := &models.PlanExecution{Status: &models.PlanExecutionStatus{
		Outputs: []*models.PlanOutputStatus{
			// plan output named differently from the step output it refers to
			{Name: "report", StepRef: &models.PlanStepOutputRef{StepID: "build", OutputName: "junit"}},
			// plan output with the same name as a step output it doesn't refer to
			{Name: "log", StepRef: &models.PlanStepOutputRef{StepID: "test", OutputName: "log"}},
		},
		Steps: []*models.PlanStepStatus{
			{ID: "build", Outputs: []*models.PlanStepOutputStatus{{Name: "junit"}, {Name: "log"}}},
			{ID: "test", Outputs: []*models.PlanStepOutputStatus{{Name: "log"}}},
		},
	}}
	var got []string
	for _, o := range collectAllOutputs(ex) {
		got = append(got, o.Scope+":"+o.Step+"/"+o.Name)
	}
	want := []string{"plan:build/report", "plan:test/log", "step:build/log"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
