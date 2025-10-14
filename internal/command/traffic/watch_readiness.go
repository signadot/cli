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
	"github.com/signadot/cli/internal/spinner"
	"github.com/signadot/cli/internal/utils/system"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
	twapi "github.com/signadot/libconnect/common/trafficwatch"
)

func waitSandboxReady(cfg *config.TrafficWatch, w io.Writer) error {
	fmt.Fprintf(w, "Waiting (up to --wait-timeout=%v) for sandbox to be ready...\n", cfg.WaitTimeout)

	params := sandboxes.NewGetSandboxParams().
		WithOrgName(cfg.Org).
		WithSandboxName(cfg.Sandbox)

	spin := spinner.Start(w, "Sandbox status")
	defer spin.Stop()

	retry := poll.
		NewPoll().
		WithTimeout(cfg.WaitTimeout)
	var failedErr error
	err := retry.Until(func() bool {
		result, err := cfg.Client.Sandboxes.GetSandbox(params, nil)
		if err != nil {
			// Keep retrying in case it's a transient error.
			spin.Messagef("error: %v", err)
			return false
		}
		sb := result.Payload
		if !sb.Status.Ready {
			if sb.Status.Reason == "ResourceFailed" {
				failedErr = errors.New(sb.Status.Message)
				return true
			}
			spin.Messagef("Not Ready: %s", sb.Status.Message)
			return false
		}
		if _, err := watchMatch(cfg, sb, true); err != nil {
			spin.Messagef("Error: %s", err.Error())
			return false
		}
		spin.StopMessagef("Ready: %s", sb.Status.Message)
		return true
	})
	if failedErr != nil {
		spin.StopFail()
		return failedErr
	}
	if err != nil {
		spin.StopFail()
		return err
	}
	return nil
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
	if _, err := watchMatch(cfg, sb, true); err != nil {
		return false, nil, err
	}
	if !sb.Status.Ready {
		return false, errors.New(sb.Status.Message), nil
	}
	return true, nil, nil
}

func ensureHasTrafficWatchClientMW(cfg *config.TrafficWatch, w io.Writer, sb *models.Sandbox) (func() error, error) {
	has, err := watchMatch(cfg, sb, false)
	if err == nil && has {
		return noUneditFunc, err
	}
	if cfg.Debug {
		fmt.Fprintf(w, "adding trafficwatch-client middleware to sandbox\n")
	}
	if err != nil {
		fmt.Fprintf(w, "WARNING: overwriting sandbox: %v\n", err)
	}

	args := []*models.SandboxesArgument{
		&models.SandboxesArgument{
			Name:  "options",
			Value: getExpectedOpts(cfg).String(),
		},
	}
	removeTrafficWatch(sb)
	sb.Spec.Middleware = append(sb.Spec.Middleware,
		&models.SandboxesMiddleware{
			Name: "trafficwatch-client",
			Args: args,
			Match: []*models.SandboxesMiddlewareMatch{
				{
					Workload: "*",
				},
			},
		})
	if sb.Spec.Labels == nil {
		sb.Spec.Labels = map[string]string{}
	}
	machineID, err := system.GetMachineID()
	if err != nil {
		return noUneditFunc, err
	}
	sb.Spec.Labels["instrumentation.signadot.com/add-trafficwatch-client"] = machineID
	params := sandboxes.NewApplySandboxParams().
		WithOrgName(cfg.Org).WithSandboxName(sb.Name).WithData(sb)
	_, err = cfg.Client.Sandboxes.ApplySandbox(params, nil)
	if err != nil {
		return noUneditFunc, err
	}
	return uneditFunc(cfg, w), nil
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

func removeTrafficWatch(sb *models.Sandbox) {
	j := 0
	for _, mw := range sb.Spec.Middleware {
		if mw.Name == "trafficwatch-client" {
			continue
		}
		sb.Spec.Middleware[j] = mw
		j++
	}
	sb.Spec.Middleware = sb.Spec.Middleware[:j]
}
