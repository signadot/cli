package trafficwatch

import (
	"log/slog"

	"github.com/signadot/libconnect/common/trafficwatch/api"
)

//const txtFormat = `id=%s time=%s normHost=%s dest=%s uri=%s method=%s proto=%s userAgent=%s`

type logMeta api.RequestMetadata

func (lm *logMeta) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("id", lm.MiddlewareRequestID),
		slog.String("time", lm.When),
		slog.String("normHost", lm.NormHost),
		slog.String("dest", lm.DestWorkload),
		slog.String("uri", lm.RequestURI),
		slog.String("method", lm.Method),
		slog.String("userAgent", lm.UserAgent),
	)
}
