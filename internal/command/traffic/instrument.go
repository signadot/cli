package traffic

import (
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/trafficwatch"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
)

func removeTrafficWatch(sb *models.Sandbox) {
	j := 0
	for _, mw := range sb.Spec.Middleware {
		if mw.Name == trafficwatch.MiddlewareName {
			continue
		}
		sb.Spec.Middleware[j] = mw
		j++
	}
	sb.Spec.Middleware = sb.Spec.Middleware[:j]
}

func noUneditFunc() error {
	return nil
}

func uneditFunc(cfg *config.TrafficWatch, w io.Writer) func() error {
	return func() error {
		params := sandboxes.NewGetSandboxParams().
			WithOrgName(cfg.Org).WithSandboxName(cfg.Sandbox)
		resp, err := cfg.Client.Sandboxes.GetSandbox(params, nil)
		if err != nil {
			return err
		}
		sb := resp.Payload
		has, err := watchMatch(cfg, sb, true)
		if err != nil {
			return err
		}
		if has {
			removeTrafficWatch(sb)
		}
		if x := sb.Spec.Labels["instrumentation.signadot.com/add-trafficwatch-client"]; x == "" {
			if !has {
				return nil
			}
		}
		delete(sb.Spec.Labels, "instrumentation.signadot.com/add-trafficwatch-client")
		applyParams := sandboxes.NewApplySandboxParams().
			WithOrgName(cfg.Org).WithSandboxName(cfg.Sandbox).WithData(sb)
		_, err = cfg.Client.Sandboxes.ApplySandbox(applyParams, nil)
		return err
	}
}
