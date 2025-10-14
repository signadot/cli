package traffic

import (
	"fmt"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/models"
	"github.com/signadot/libconnect/common/trafficwatch/api"
)

func watchMatch(cfg *config.TrafficWatch, sb *models.Sandbox, applied bool) (bool, error) {
	count := 0
	for _, mw := range sb.Spec.Middleware {
		if mw.Name != "trafficwatch-client" {
			continue
		}
		if count > 0 {
			return false, fmt.Errorf("sandbox %s has multiple traffic-watch-client middlewares", sb.Name)
		}
		count++
		wantOpts := getExpectedOpts(cfg)
		if len(mw.Args) != 1 {
			return false, fmt.Errorf("sandbox %s has traffic-watch-client middleware configured differently than expected: too many args (%d)", sb.Name, len(mw.Args))
		}
		// NB the middleware could be configured consistently there is no way to know
		mwa := mw.Args[0]
		if mwa.Name != "options" {
			return false, fmt.Errorf("sandbox %s has traffic-watch-client middleware configured differently than expected: unexpected arg %q", sb.Name, mwa.Name)
		}
		if mwa.Value != wantOpts.String() {
			return false, fmt.Errorf("sandbox %s has traffic-watch-client middleware configured differently than expected: wanted options %s got %s", sb.Name, wantOpts, mwa.Value)
		}
	}
	if count == 1 {
		return true, nil
	}
	if applied {
		return false, fmt.Errorf("sandbox %s no longer has traffic-watch-client middleware", sb.Name)
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
