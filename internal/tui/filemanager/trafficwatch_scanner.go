package filemanager

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/signadot/libconnect/common/trafficwatch/api"
)

type TrafficWatchScanner struct {
	cfg    *TrafficWatchScannerConfig
	offset int64

	resumeCh chan struct{}
	closeCh  chan struct{}
}

func NewTrafficWatchScanner(cfg *TrafficWatchScannerConfig) *TrafficWatchScanner {
	return &TrafficWatchScanner{
		cfg:      cfg,
		offset:   0,
		resumeCh: make(chan struct{}),
		closeCh:  make(chan struct{}),
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
	file, err := os.Open(tw.cfg.mainMetaPath)
	if err != nil {
		return err
	}

	offset, err := file.Seek(tw.offset, io.SeekStart)
	if err != nil {
		return err
	}
	tw.offset = offset

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
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		metaRequest := api.RequestMetadata{}
		err = json.Unmarshal(line, &metaRequest)
		if err != nil {
			continue
		}
		tw.offset += int64(len(line)) + 1

		if metaRequest.DestWorkload == "" {
			continue
		}

		tw.cfg.onNewLine(metaRequest)
	}

	if err := scanner.Err(); err != nil {
		return
	}
}
