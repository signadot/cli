package plan

import (
	"io"

	"github.com/signadot/cli/internal/command/planshared"
	"github.com/signadot/go-sdk/models"
)

func printPlanDetails(out io.Writer, p *models.RunnablePlan) error {
	return planshared.PrintPlanDetails(out, p)
}
