package smarttest

import (
	"testing"

	"github.com/signadot/go-sdk/models"
)

func TestUnsuccessfulTestsError(t *testing.T) {
	tx := func(ph models.TestexecutionsPhase) *models.TestExecution {
		return &models.TestExecution{Status: &models.TestExecutionStatus{Phase: ph}}
	}
	cases := []struct {
		name    string
		txs     []*models.TestExecution
		wantErr bool
	}{
		{"none", nil, false},
		{"succeeded", []*models.TestExecution{tx(models.TestexecutionsPhaseSucceeded)}, false},
		{"failed", []*models.TestExecution{tx(models.TestexecutionsPhaseSucceeded), tx(models.TestexecutionsPhaseFailed)}, true},
		{"canceled", []*models.TestExecution{tx(models.TestexecutionsPhaseCanceled)}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := unsuccessfulTestsError(c.txs)
			if (err != nil) != c.wantErr {
				t.Fatalf("got err %v, wantErr %v", err, c.wantErr)
			}
		})
	}
}
