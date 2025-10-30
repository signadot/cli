package filemanager

import (
	"path/filepath"
)

func GetSourceRequestPath(recordDir, requestID string) string {
	return filepath.Join(recordDir, requestID, "request")
}

func GetSourceResponsePath(recordDir, requestID string) string {
	return filepath.Join(recordDir, requestID, "response")
}
