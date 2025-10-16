package override

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/poll"
	"github.com/signadot/go-sdk/models"
)

func monitorSandbox(ctx context.Context, cfg *config.LocalOverrideCreate,
	sandbox *models.Sandbox, overrideName string) {
	poll.NewPoll().Readiness(ctx, 5*time.Second, func() (bool, error, error) {
		return ckMatch(cfg, sandbox, overrideName)
	})
}

func ckMatch(cfg *config.LocalOverrideCreate, sb *models.Sandbox, override string) (bool, error, error) {
	obs, err := getSandbox(cfg)
	if err != nil {
		return false, nil, err
	}
	ready := obs.Status.Ready
	if err := ckAllMiddleware(sb.Spec.Middleware, obs.Spec.Middleware, sb.Name, override); err != nil {
		return ready, nil, err
	}
	if err := ckForwards(sb.Spec.Routing, obs.Spec.Routing, sb.Name, override); err != nil {
		return ready, nil, err
	}

	if !ready {
		return false, fmt.Errorf("sandbox %s is not ready to accept requests", sb.Name), nil
	}
	return true, nil, nil

}

func ckAllMiddleware(desMWs, obsMWs []*models.SandboxesMiddleware, sbName, oName string) error {
	desOMWs := getOverrideMWs(desMWs, oName)
	obsOMWs := getOverrideMWs(obsMWs, oName)
	if len(obsOMWs) < len(desOMWs) {
		return fmt.Errorf("sandbox %s is missing override middleware", sbName)
	}
	if len(obsOMWs) > len(desOMWs) {
		return fmt.Errorf("sandbox %s has interfering override middleware", sbName)
	}
	for i := range desOMWs {
		obs, des := obsOMWs[i], desOMWs[i]
		if err := ckMiddleware(des, obs, sbName, oName); err != nil {
			return err
		}
	}
	return nil
}

func ckMiddleware(des, obs *models.SandboxesMiddleware, sbName, oName string) error {
	if len(obs.Args) < len(des.Args) {
		return fmt.Errorf("sandbox %s missing override middleware arguments", sbName)
	}
	// args
	argMap := map[string]*models.SandboxesArgument{}
	for _, da := range des.Args {
		argMap[da.Name] = da
	}
	for _, oa := range obs.Args {
		da := argMap[oa.Name]
		if da == nil {
			return argConflict(sbName, oa.Name)
		}
		if da.Value != oa.Value {
			return argConflict(sbName, oa.Name)
		}
		if (da.ValueFrom == nil) != (oa.ValueFrom == nil) {
			return argConflict(sbName, oa.Name)
		}
		if da.ValueFrom == nil {
			continue
		}
		if da.ValueFrom.Forward != oa.ValueFrom.Forward {
			return argConflict(sbName, oa.Name)
		}
	}
	// matches
	if len(obs.Match) != len(des.Match) {
		return fmt.Errorf("sandbox %s has unexpected match criteria", sbName)
	}
	mMap := map[string]bool{}
	for _, match := range des.Match {
		mMap[match.Workload] = true
	}
	for _, match := range obs.Match {
		if mMap[match.Workload] {
			continue
		}
		return fmt.Errorf("sandbox %s has unexpected match criteria", sbName)
	}
	return nil
}

func argConflict(sbName, argName string) error {
	return fmt.Errorf("sandbox %s has conflicting middleware argument %q", sbName, argName)
}

func getOverrideMWs(mws []*models.SandboxesMiddleware, oName string) []*models.SandboxesMiddleware {
	res := []*models.SandboxesMiddleware{}
	for _, mw := range mws {
		if mw.Name != "override" {
			continue
		}
		found := false
		for _, arg := range mw.Args {
			if arg.Name != "overrideHost" {
				continue
			}
			if arg.ValueFrom == nil {
				continue
			}
			if arg.ValueFrom.Forward == oName {
				found = true
				break
			}
		}
		if found {
			res = append(res, mw)
		}
	}
	return res
}

func ckForwards(des, obs *models.SandboxesRouting, sbName, oName string) error {
	if des == nil {
		return fmt.Errorf("sandbox %s has no desired forwards", sbName)
	}
	if obs == nil {
		return fmt.Errorf("sandbox %s has no forwards", sbName)
	}
	desFwd, desLogFwd := getOFwds(des.Forwards, oName)
	obsFwd, obsLogFwd := getOFwds(obs.Forwards, oName)

	if (desFwd == nil) != (obsFwd == nil) {
		return fmt.Errorf("sandbox %s no longer has matching forwards", sbName)
	}
	if desFwd != nil {
		if !reflect.DeepEqual(desFwd, obsFwd) {
			return fmt.Errorf("sandbox %s no longer has matching forwards", sbName)
		}
	}
	if (desLogFwd == nil) != (obsLogFwd == nil) {
		return fmt.Errorf("sandbox %s no longer has matching log forwards", sbName)
	}
	if desLogFwd != nil {
		if !reflect.DeepEqual(desLogFwd, obsLogFwd) {
			return fmt.Errorf("sandbox %s no longer has matching log forwards", sbName)
		}
	}
	return nil
}

func getOFwds(fwds []*models.SandboxesForward, oName string) (mw, log *models.SandboxesForward) {
	for _, fwd := range fwds {
		if fwd.Name == oName {
			mw = fwd
			continue
		}
		if fwd.Name == oName+"-log" {
			log = fwd
		}
		if mw != nil && fwd != nil {
			break
		}
	}
	return
}
