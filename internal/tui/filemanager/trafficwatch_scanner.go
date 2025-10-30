package filemanager

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/signadot/cli/internal/config"
)

type TrafficWatchScanner struct {
	cfg    *TrafficWatchScannerConfig
	offset int64

	resumeCh chan struct{}
	closeCh  chan struct{}

	pendingRequests map[string]*RequestMetadata
}

func NewTrafficWatchScanner(cfg *TrafficWatchScannerConfig) *TrafficWatchScanner {
	return &TrafficWatchScanner{
		cfg:             cfg,
		offset:          0,
		resumeCh:        make(chan struct{}),
		closeCh:         make(chan struct{}),
		pendingRequests: make(map[string]*RequestMetadata),
	}
}

func (tw *TrafficWatchScanner) Resume() {
	select {
	case tw.resumeCh <- struct{}{}:
	default:
	}
}

func (tw *TrafficWatchScanner) Close() {
	select {
	case <-tw.closeCh:
		return
	default:
		close(tw.closeCh)
	}
}

func (tw *TrafficWatchScanner) Start(ctx context.Context) error {
	// Start continuous monitoring with ticker
	go tw.monitorWithTicker(ctx)

	return nil
}

func (tw *TrafficWatchScanner) monitorWithTicker(ctx context.Context) {
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
	file, err := os.Open(tw.cfg.mainMetaPath)
	if err != nil {
		tw.cfg.onNewLine(&LineMessage{
			MessageType: MessageTypeStatusNoStarted,
			Error:       err,
		})
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
	if tw.cfg.recordsFormat == config.OutputFormatYAML {
		scanner.Split(tw.splitYAML)
	}

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		metaRequest := &RequestMetadata{}

		switch tw.cfg.recordsFormat {
		case config.OutputFormatJSON:
			err = json.Unmarshal(line, metaRequest)
			if err != nil {
				continue
			}
			tw.offset += int64(len(line)) + 1 // \n

		case config.OutputFormatYAML:
			err = yaml.Unmarshal(line, metaRequest)
			if err != nil {
				continue
			}

			tw.offset += int64(len(line)) + 4 // --- + \n
		}

		if err != nil {
			continue
		}

		tw.handleMessageRequest(metaRequest)
	}

	if err := scanner.Err(); err != nil {
		return
	}
}

func (tw *TrafficWatchScanner) handleMessageRequest(request *RequestMetadata) {
	if !request.DoneAt.IsZero() {
		pendingRequest, ok := tw.pendingRequests[request.MiddlewareRequestID]
		if !ok {
			return
		}

		pendingRequest.DoneAt = request.DoneAt

		response, _ := tw.loadResponseFile(pendingRequest)
		if response != nil {
			switch response.Header.Get("Content-Type") {
			case "application/grpc":
				pendingRequest.Protocol = ProtocolGRPC
			default:
				pendingRequest.Protocol = ProtocolHTTP
			}
		}

		tw.cfg.onNewLine(&LineMessage{
			MessageType: MessageTypeData,
			Data:        pendingRequest,
		})

		delete(tw.pendingRequests, request.MiddlewareRequestID)
		return
	}

	if _, ok := tw.pendingRequests[request.MiddlewareRequestID]; !ok {
		tw.pendingRequests[request.MiddlewareRequestID] = request
	}
}

func (tw *TrafficWatchScanner) loadResponseFile(request *RequestMetadata) (*http.Response, error) {
	requestFile, err := os.Open(GetSourceResponsePath(tw.cfg.recordDir, request.MiddlewareRequestID))
	if err != nil {
		return nil, err
	}
	defer requestFile.Close()

	return http.ReadResponse(bufio.NewReader(requestFile), &http.Request{})
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
