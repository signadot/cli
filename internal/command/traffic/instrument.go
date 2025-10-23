package traffic

import (
	"context"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	sbmgr "github.com/signadot/cli/internal/locald/sandboxmanager"
	"github.com/signadot/cli/internal/trafficwatch"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/cli/internal/utils/system"
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

func noOpUndo() error {
	return nil
}

func mkUndo(cfg *config.TrafficWatch, w io.Writer) func() error {
	return func() error {
		ctx := context.Background()
		sb, err := utils.GetSandbox(ctx, cfg.API, cfg.Sandbox)
		if err != nil {
			return err
		}
		has, err := watchMatch(cfg, sb, true)
		if err != nil {
			return err
		}
		if has {
			removeTrafficWatch(sb)
		}
		if x := sb.Spec.Labels[trafficwatch.InstrumentationKey]; x == "" {
			if !has {
				return nil
			}
		}
		delete(sb.Spec.Labels, trafficwatch.InstrumentationKey)

		printTWProgress(w, fmt.Sprintf("Removing %s middleware from sandbox %s",
			trafficwatch.MiddlewareName, cfg.Sandbox))
		return applyWithLocal(ctx, cfg, sb)
	}
}

func applyWithLocal(ctx context.Context, cfg *config.TrafficWatch,
	sb *models.Sandbox) error {
	hasLocal := false
	if sb.Spec.Routing != nil && len(sb.Spec.Routing.Forwards) != 0 {
		hasLocal = true
	}
	if len(sb.Spec.Local) != 0 {
		hasLocal = true
	}
	if hasLocal {
		_, err := sbmgr.ValidateSandboxManager(sb.Spec.Cluster)
		if err != nil {
			return err
		}
		machineID, err := system.GetMachineID()
		if err != nil {
			return err
		}
		sb.Spec.LocalMachineID = machineID
	}

	applyParams := sandboxes.NewApplySandboxParams().
		WithContext(ctx).
		WithOrgName(cfg.Org).
		WithSandboxName(cfg.Sandbox).
		WithData(sb)

	_, err := cfg.Client.Sandboxes.ApplySandbox(applyParams, nil)
	return err
}
