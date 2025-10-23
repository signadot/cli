package traffic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/poll"
	"github.com/signadot/cli/internal/trafficwatch"
	"github.com/signadot/cli/internal/utils/system"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
	twapi "github.com/signadot/libconnect/common/trafficwatch"
)

func ensureHasTrafficWatchClientMW(ctx context.Context, cfg *config.TrafficWatch,
	w io.Writer, sb *models.Sandbox) (func() error, error) {
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
	machineID, err := system.GetMachineID()
	if err != nil {
		return noOpUndo, err
	}
	sb.Spec.Labels[fmt.Sprintf("instrumentation.signadot.com/add-%s", trafficwatch.MiddlewareName)] = machineID
	if err := applyWithLocal(ctx, cfg, sb); err != nil {
		return noOpUndo, err
	}
	return mkUndo(cfg, w), nil
}

func readyLoop(ctx context.Context, log *slog.Logger, tw *twapi.TrafficWatch, readiness poll.Readiness) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tw.Close:
			return
		case <-ticker.C:
			if err := readiness.Fatal(); err != nil {
				log.Error("exiting due to problem with sandbox", "error", err)
				tw.JustClose()
			}
			for err := readiness.Warn(); err != nil; err = readiness.Warn() {
				log.Warn("sandbox not ready", "error", err)
			}
		}
	}
}

func ckReady(cfg *config.TrafficWatch) (ready bool, warn, fatal error) {
	params := sandboxes.NewGetSandboxParams().
		WithOrgName(cfg.Org).WithSandboxName(cfg.Sandbox)
	resp, err := cfg.Client.Sandboxes.GetSandbox(params, nil)
	if err != nil {
		return false, err, nil
	}
	sb := resp.Payload
	has, err := watchMatch(cfg, sb, true)
	if !has {
		return sb.Status.Ready, nil, fmt.Errorf("sandbox no longer has %s middleware", trafficwatch.MiddlewareName)
	}
	if !cfg.NoInstrument && err != nil {
		return sb.Status.Ready, nil, fmt.Errorf("sandbox no longer has %s middleware", trafficwatch.MiddlewareName)
	}
	// either middleware matches or we aren't trying to instrument and allow
	// out of band mw edits
	if !sb.Status.Ready {
		return false, errors.New(sb.Status.Message), nil
	}
	return true, nil, nil
}
