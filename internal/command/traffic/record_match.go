package traffic

import (
	"fmt"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/trafficwatch"
	"github.com/signadot/go-sdk/models"
	"github.com/signadot/libconnect/common/trafficwatch/api"
)

func watchMatch(cfg *config.TrafficWatch, sb *models.Sandbox, applied bool) (bool, error) {
	count := 0
	for _, mw := range sb.Spec.Middleware {
		if mw.Name != trafficwatch.MiddlewareName {
			continue
		}
		if count > 0 {
			return false, fmt.Errorf("sandbox %s has multiple %s middlewares", sb.Name, trafficwatch.MiddlewareName)
		}
		count++
		wantOpts := getExpectedOpts(cfg)
		if len(mw.Args) != 1 {
			return false, fmt.Errorf("sandbox %s has %s middleware configured differently than expected: too many args (%d)", sb.Name, trafficwatch.MiddlewareName, len(mw.Args))
		}
		// NB the middleware could be configured consistently there is no way to know
		mwa := mw.Args[0]
		if mwa.Name != "options" {
			return false, fmt.Errorf("sandbox %s has %s middleware configured differently than expected: unexpected arg %q", sb.Name, trafficwatch.MiddlewareName, mwa.Name)
		}
		if mwa.Value != wantOpts.String() {
			return false, fmt.Errorf("sandbox %s has %s middleware configured differently than expected: wanted options %s got %s", sb.Name, trafficwatch.MiddlewareName, wantOpts, mwa.Value)
		}
		if len(mw.Match) != 1 {
			return false, fmt.Errorf("sandbox %s has %s middleware configured differently than expected: match differs", sb.Name, trafficwatch.MiddlewareName)
		}
		mwMatch := mw.Match[0]
		if mwMatch == nil || mwMatch.Workload != "*" {
			return false, fmt.Errorf("sandbox %s has %s middleware configured differently than expected: match differs", sb.Name, trafficwatch.MiddlewareName)
		}
	}
	if count == 1 {
		return true, nil
	}
	if applied {
		return false, fmt.Errorf("sandbox %s no longer has %s middleware", sb.Name, trafficwatch.MiddlewareName)
	}
	return false, nil
}

func getExpectedOpts(cfg *config.TrafficWatch) *api.WatchOptions {
	if cfg.Short {
		return api.WatchShort()
	}
	if cfg.HeadersOnly {
		return api.WatchTruncate(0)
	}
	return api.WatchAll()
}
