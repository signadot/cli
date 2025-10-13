package trafficwatch

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/signadot/cli/internal/config"
)

func encodeReqDones(rdC <-chan string, log *slog.Logger, fn func(*slog.Logger, *reqDone), encs ...*json.Encoder) {
	for id := range rdC {
		rd := newReqDone(id)
		for _, enc := range encs {
			err := enc.Encode(rd)
			if err == nil {
				continue
			}
			log.Warn("error encoding request metadata", "error", err)
		}
		if fn != nil {
			fn(log, rd)
		}
	}
}

func handleDir(cfg *config.TrafficWatch) func(log *slog.Logger, reqDone *reqDone) {
	return func(log *slog.Logger, reqDone *reqDone) {
		reqDir := filepath.Join(cfg.ToDir, reqDone.ID, "meta.json")
		d, err := os.ReadFile(reqDir)
		if err != nil {
			log.Warn("error reading", "path", reqDir, "error", err)
			return
		}
		x := map[string]any{}
		if err := json.Unmarshal(d, &x); err != nil {
			log.Warn("unable to decode json", "path", reqDir, "error", err)
			return
		}
		x["doneAt"] = reqDone.DoneAt
		d, err = json.MarshalIndent(x, "", "  ")
		if err != nil {
			log.Warn("unable to encode json", "path", reqDir, "error", err)
			return
		}
		f, err := os.OpenFile(reqDir, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Warn("unable to open meta file", "path", reqDir, "error", err)
			return
		}
		defer f.Close()
		if _, err := f.Write(d); err != nil {
			log.Warn("unable to write meta file", "path", reqDir, "error", err)
			return
		}
	}
}

type reqDone struct {
	ID     string `json:"middlewareRequestID"`
	DoneAt string `json:"doneAt"`
}

func newReqDone(id string) *reqDone {
	return &reqDone{
		ID:     id,
		DoneAt: time.Now().Format(time.RFC3339Nano),
	}
}
