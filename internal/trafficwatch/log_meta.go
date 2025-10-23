package trafficwatch

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/libconnect/common/trafficwatch/api"
)

type logMeta api.RequestMetadata

func (lm *logMeta) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("id", lm.MiddlewareRequestID),
		slog.String("time", lm.When),
		slog.String("dest", lm.DestWorkload),
		slog.String("uri", lm.RequestURI),
		slog.String("method", lm.Method),
		slog.String("userAgent", lm.UserAgent),
	)
}

type metaEncoder interface {
	Encode(v any) error
	metaEncoder()
}

type mEnc struct {
	sync.Mutex
	j       *json.Encoder
	n       int
	yWriter io.Writer
}

func (e *mEnc) metaEncoder() {}

func (e *mEnc) Encode(v any) error {
	e.Lock()
	defer e.Unlock()
	defer func() { e.n++ }()
	if e.j != nil {
		return e.j.Encode(v)
	}
	if e.n != 0 {
		_, err := e.yWriter.Write([]byte("---\n"))
		if err != nil {
			return err
		}
	}
	return print.RawK8SYAML(e.yWriter, v)
}

func getMetaEncoder(w io.Writer, cfg *config.TrafficWatch) *mEnc {
	switch cfg.OutputFormat {
	case config.OutputFormatJSON, config.OutputFormatDefault:
		return &mEnc{j: json.NewEncoder(w)}
	case config.OutputFormatYAML:
		return &mEnc{yWriter: w}
	default:
		panic(fmt.Sprintf("unknown output format %q", cfg.OutputFormat))
	}
}
