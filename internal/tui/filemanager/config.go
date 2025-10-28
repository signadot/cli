package filemanager

import (
	"fmt"
	"path/filepath"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/libconnect/common/trafficwatch/api"
)

type MessageType string

const (
	MessageTypeData            MessageType = "data"
	MessageTypeStatusNoStarted MessageType = "target_not_found"
)

type LineMessage struct {
	MessageType MessageType
	Data        *api.RequestMetadata
	Error       error
}

type OnNewLineCallback func(lineMessage *LineMessage)
type TrafficWatchScannerConfig struct {
	recordDir     string
	recordsFormat config.OutputFormat

	mainMetaPath string

	onNewLine OnNewLineCallback
}

func (cfg *TrafficWatchScannerConfig) GetRecordDir() string {
	return cfg.recordDir
}

func NewTrafficWatchScannerConfig(opts ...func(*TrafficWatchScannerConfig)) (*TrafficWatchScannerConfig, error) {
	cfg := &TrafficWatchScannerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.recordDir == "" {
		return nil, fmt.Errorf("recordDir is required")
	}

	if cfg.recordsFormat == "" {
		return nil, fmt.Errorf("recordsFormat is required")
	}

	// TODO: Validate the recordDir contains the expected files (meta.jsons or meta.yamls)
	// TODO: Validate recordDir only contain one of the formats (meta.jsons or meta.yamls)

	var metaName string
	switch cfg.recordsFormat {
	case config.OutputFormatJSON:
		metaName = "meta.jsons"
	case config.OutputFormatYAML:
		metaName = "meta.yamls"
	}

	mainMetaPath := filepath.Join(cfg.recordDir, metaName)
	cfg.mainMetaPath = mainMetaPath

	return cfg, nil
}

func WithRecordDir(recordDir string) func(*TrafficWatchScannerConfig) {
	return func(config *TrafficWatchScannerConfig) {
		config.recordDir = recordDir
	}
}

func WithRecordsFormat(recordsFormat config.OutputFormat) func(*TrafficWatchScannerConfig) {
	return func(config *TrafficWatchScannerConfig) {
		config.recordsFormat = recordsFormat
	}
}

func WithOnNewLine(onNewLine OnNewLineCallback) func(*TrafficWatchScannerConfig) {
	return func(config *TrafficWatchScannerConfig) {
		config.onNewLine = onNewLine
	}
}

func WithMainMetaPath(mainMetaPath string) func(*TrafficWatchScannerConfig) {
	return func(config *TrafficWatchScannerConfig) {
		config.mainMetaPath = mainMetaPath
	}
}
