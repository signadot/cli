package plan

import (
	"github.com/signadot/cli/internal/config"
	plantags "github.com/signadot/go-sdk/client/plan_tags"
	"github.com/signadot/go-sdk/models"
)

// tagPlan tags a plan with the given name. Caller must have already called InitAPIConfig.
func tagPlan(cfg *config.Plan, planID, tagName string) error {
	params := plantags.NewPutPlanTagParams().
		WithOrgName(cfg.Org).
		WithPlanTagName(tagName).
		WithData(&models.PlanTagSpec{
			PlanID: planID,
		})
	_, err := cfg.Client.PlanTags.PutPlanTag(params, nil)
	return err
}
