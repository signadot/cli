package trafficwatch

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/signadot/cli/internal/config"
)

func waitLogged(logged, done <-chan string) chan string {
	res := make(chan string)
	go func() {
		logMap, doneMap := map[string]bool{}, map[string]bool{}
		for {
			select {
			case id, ok := <-logged:
				if !ok {
					if done == nil {
						return
					}
					logged = nil
				} else {
					if doneMap[id] {
						res <- id
						delete(doneMap, id)
					} else {
						logMap[id] = true
					}
				}
			case id, ok := <-done:
				if !ok {
					if logged == nil {
						return
					}
					done = nil
				} else {
					if logMap[id] {
						res <- id
						delete(logMap, id)
					} else {
						doneMap[id] = true
					}
				}
			}
		}
	}()
	return res
}

func encodeReqDones(rdC <-chan string, log *slog.Logger, fn func(*slog.Logger, *reqDone), enc metaEncoder) {
	for id := range rdC {
		rd := newReqDone(id)
		log.Info("request-done", "request", rd)
		if enc != nil {
			err := enc.Encode(rd)
			if err != nil {
				log.Warn("error encoding request metadata", "error", err)
			}
		}
		if fn != nil {
			fn(log, rd)
		}
	}
}

func handleDir(cfg *config.TrafficWatch) func(log *slog.Logger, reqDone *reqDone) {
	suffix := ".json"
	if cfg.OutputFormat == config.OutputFormatYAML {
		suffix = ".yaml"
	}
	return func(log *slog.Logger, reqDone *reqDone) {
		reqMetaPath := filepath.Join(cfg.OutputDir, reqDone.ID, "meta"+suffix)
		d, err := os.ReadFile(reqMetaPath)
		if err != nil {
			log.Warn("error reading", "path", reqMetaPath, "error", err)
			return
		}
		x := map[string]any{}
		if err := yaml.Unmarshal(d, &x); err != nil {
			log.Warn("unable to decode json", "path", reqMetaPath, "error", err)
			return
		}
		x["doneAt"] = reqDone.DoneAt
		f, err := os.OpenFile(reqMetaPath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Warn("unable to open meta file", "path", reqMetaPath, "error", err)
			return
		}
		defer f.Close()
		enc := getMetaEncoder(f, cfg)
		if enc.j != nil {
			enc.j.SetIndent("", "  ")
		}
		if err := enc.Encode(x); err != nil {
			log.Warn("unable to write meta file", "path", reqMetaPath, "error", err)
			return
		}
	}
}

type reqDone struct {
	ID     string `json:"middlewareRequestID"`
	DoneAt string `json:"doneAt"`
}

func (rd *reqDone) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("id", rd.ID),
		slog.String("doneAt", rd.DoneAt),
	)
}

func newReqDone(id string) *reqDone {
	return &reqDone{
		ID:     id,
		DoneAt: time.Now().Format(time.RFC3339Nano),
	}
}
