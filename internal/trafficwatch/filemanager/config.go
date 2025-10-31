package filemanager

import (
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/libconnect/common/trafficwatch/api"
)

type Protocol string

const (
	ProtocolHTTP Protocol = "http"
	ProtocolGRPC Protocol = "gRPC"
)

type RequestMetadata struct {
	api.RequestMetadata

	DoneAt   time.Time `json:"doneAt"`
	Protocol Protocol
}

type OnRequest func(reqMeta *RequestMetadata)
type OnError func(err error)

type ScannerConfig struct {
	TrafficDir string
	Format     config.OutputFormat
	OnRequest  OnRequest
	OnError    OnError
}
