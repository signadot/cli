package trafficwatch

import "github.com/signadot/cli/internal/config"

const (
	MiddlewareName     = "trafficwatch"
	DefaultDirRelative = "traffic/watch"
	InstrumentationKey = "instrumentation.signadot.com/add-" + MiddlewareName
)

func FormatSuffix(cfg *config.TrafficWatch) string {
	if cfg.OutputFormat == config.OutputFormatYAML {
		return ".yaml"
	}
	return ".json"
}

func StreamFormatSuffix(cfg *config.TrafficWatch) string {
	return FormatSuffix(cfg) + "s"
}
