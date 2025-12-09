package traffic

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/devbox"
	sbmgr "github.com/signadot/cli/internal/locald/sandboxmanager"
	"github.com/signadot/cli/internal/trafficwatch"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
)

type undoFunc func(ctx context.Context, w io.Writer) error

func ensureTrafficWatchMW(ctx context.Context, cfg *config.TrafficWatch,
	w io.Writer, sb *models.Sandbox) (undoFunc, error) {
	if cfg.NoInstrument {
		return noOpUndo, nil
	}
	has, err := watchMatch(cfg, sb, cfg.NoInstrument)
	if err == nil && has {
		return noOpUndo, err
	}
	printTWProgress(w, fmt.Sprintf("Applying %s middleware to sandbox %s",
		trafficwatch.MiddlewareName, cfg.Sandbox))
	if err != nil {
		fmt.Fprintf(w, "WARNING: overwriting sandbox: %v\n", err)
	}

	mw := mwSpec(cfg)
	removeTrafficWatch(sb)
	sb.Spec.Middleware = append(sb.Spec.Middleware, mw)
	if sb.Spec.Labels == nil {
		sb.Spec.Labels = map[string]string{}
	}
	devboxID, err := devbox.GetID(ctx, cfg.API, false, "")
	if err != nil {
		return noOpUndo, err
	}
	sb.Spec.Labels[trafficwatch.InstrumentationKey] = devboxID
	if err := applyWithLocal(ctx, cfg, sb); err != nil {
		return noOpUndo, err
	}
	return mkUndo(cfg), nil
}

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

func noOpUndo(context.Context, io.Writer) error {
	return nil
}

func mkUndo(cfg *config.TrafficWatch) undoFunc {
	return func(ctx context.Context, out io.Writer) error {
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

		printTWProgress(out, fmt.Sprintf("Removing %s middleware from sandbox %s",
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
		connectStatus, err := sbmgr.ValidateSandboxManager(sb.Spec.Cluster)
		if err != nil {
			return err
		}
		if connectStatus.DevboxSession == nil {
			return errors.New("no devbox session")
		}
		sb.Spec.Connection = &models.Connection{
			DevboxID: connectStatus.DevboxSession.DevboxId,
		}
	}
	// remove deprecated
	sb.Spec.Endpoints = nil
	for _, f := range sb.Spec.Forks {
		f.Endpoints = nil
	}

	applyParams := sandboxes.NewApplySandboxParams().
		WithContext(ctx).
		WithOrgName(cfg.Org).
		WithSandboxName(cfg.Sandbox).
		WithData(sb)

	_, err := cfg.Client.Sandboxes.ApplySandbox(applyParams, nil)
	return err
}
