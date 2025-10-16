package traffic

import (
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/trafficwatch"
	"github.com/signadot/go-sdk/models"
)

func mwSpec(cfg *config.TrafficWatch) *models.SandboxesMiddleware {
	args := []*models.SandboxesArgument{
		&models.SandboxesArgument{
			Name:  "options",
			Value: getExpectedOpts(cfg).String(),
		},
	}
	return &models.SandboxesMiddleware{
		Name: trafficwatch.MiddlewareName,
		Args: args,
		Match: []*models.SandboxesMiddlewareMatch{
			{
				Workload: "*",
			},
		},
	}
}
