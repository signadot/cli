package filemanager

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/signadot/cli/internal/config"
)

type TrafficWatchScanner struct {
	ScannerConfig

	path   string
	offset int64

	resumeCh chan struct{}
	closeCh  chan struct{}
	closeOnce sync.Once

	pendingRequests map[string]*RequestMetadata
}

func NewTrafficWatchScanner(cfg *ScannerConfig) (*TrafficWatchScanner, error) {
	if cfg.TrafficDir == "" {
		return nil, fmt.Errorf("trafficDir is required")
	}

	if cfg.Format == "" {
		return nil, fmt.Errorf("format is required")
	}

	var metaName string
	switch cfg.Format {
	case config.OutputFormatJSON:
		metaName = "meta.jsons"
	case config.OutputFormatYAML:
		metaName = "meta.yamls"
	default:
		return nil, fmt.Errorf("invalid format")
	}
	metaPath := filepath.Join(cfg.TrafficDir, metaName)

	return &TrafficWatchScanner{
		ScannerConfig: *cfg,

		offset:   0,
		path:     metaPath,
		resumeCh: make(chan struct{}),
		closeCh:  make(chan struct{}),

		pendingRequests: make(map[string]*RequestMetadata),
	}, nil
}

func (tw *TrafficWatchScanner) Init() []*RequestMetadata {
	onReqOrig := tw.OnRequest
	defer func() {
		tw.OnRequest = onReqOrig
	}()

	var requests []*RequestMetadata
	tw.OnRequest = func(reqMeta *RequestMetadata) {
		requests = append(requests, reqMeta)
	}
	tw.checkForNewContent()
	return requests
}

func (tw *TrafficWatchScanner) Start(ctx context.Context) error {
	// Start continuous monitoring with ticker
	go tw.run(ctx)
	return nil
}

func (tw *TrafficWatchScanner) Resume() {
	select {
	case tw.resumeCh <- struct{}{}:
	default:
	}
}

func (tw *TrafficWatchScanner) Close() {
	tw.closeOnce.Do(func() {
		close(tw.closeCh)
	})
}

func (tw *TrafficWatchScanner) run(ctx context.Context) {
	// Create a ticker that checks for file changes every 500ms
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tw.closeCh:
			return
		case <-ticker.C:
			// Check for new content every tick
			tw.checkForNewContent()
		case <-tw.resumeCh:
			// Manual resume signal - check immediately
			tw.checkForNewContent()
		}
	}
}

func (tw *TrafficWatchScanner) checkForNewContent() {
	file, err := os.Open(tw.path)
	if err != nil {
		if tw.OnError != nil {
			tw.OnError(err)
		}
		return
	}
	defer file.Close()

	// Seek to our last known position
	_, err = file.Seek(tw.offset, io.SeekStart)
	if err != nil {
		return
	}

	// Read new content
	scanner := bufio.NewScanner(file)

	// If the records format is YAML, use the custom split function.
	// For JSON, use the default split function.
	if tw.Format == config.OutputFormatYAML {
		scanner.Split(tw.splitYAML)
	}

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var reqMeta RequestMetadata
		switch tw.Format {
		case config.OutputFormatJSON:
			err = json.Unmarshal(line, &reqMeta)
			if err != nil {
				continue
			}
			tw.offset += int64(len(line)) + 1 // \n

		case config.OutputFormatYAML:
			err = yaml.Unmarshal(line, &reqMeta)
			if err != nil {
				continue
			}

			tw.offset += int64(len(line)) + 4 // --- + \n
		}
		if err != nil {
			continue
		}

		tw.handleRequestEvent(&reqMeta)
	}

	if err := scanner.Err(); err != nil {
		return
	}
}

func (tw *TrafficWatchScanner) handleRequestEvent(reqEvent *RequestMetadata) {
	reqID := reqEvent.MiddlewareRequestID

	if !reqEvent.DoneAt.IsZero() {
		// this a request done event
		reqMeta, ok := tw.pendingRequests[reqID]
		if !ok {
			return
		}

		// set the done at time
		reqMeta.DoneAt = reqEvent.DoneAt

		// set the protocol
		resp, _ := LoadHttpResponse(GetSourceResponsePath(tw.TrafficDir, reqID))
		if resp != nil {
			switch resp.Header.Get("Content-Type") {
			case "application/grpc":
				reqMeta.Protocol = ProtocolGRPC
			default:
				reqMeta.Protocol = ProtocolHTTP
			}
		}

		if tw.OnRequest != nil {
			tw.OnRequest(reqMeta)
		}
		delete(tw.pendingRequests, reqID)
		return
	}

	// this is a request start event
	if _, ok := tw.pendingRequests[reqID]; !ok {
		tw.pendingRequests[reqID] = reqEvent
	}
}

func (tw *TrafficWatchScanner) splitYAML(data []byte, atEOF bool) (advance int, token []byte, err error) {
	sep := []byte("---\n")

	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if i := bytes.Index(data, sep); i >= 0 {
		return i + len(sep), data[:i], nil
	}

	// At EOF: emit any remaining non-empty data
	if atEOF {
		return len(data), data, nil
	}

	return 0, nil, nil // request more data
}
