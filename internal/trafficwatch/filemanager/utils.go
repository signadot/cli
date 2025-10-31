package filemanager

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func GetSourceRequestPath(recordDir, requestID string) string {
	return filepath.Join(recordDir, requestID, "request")
}

func GetSourceResponsePath(recordDir, requestID string) string {
	return filepath.Join(recordDir, requestID, "response")
}

func LoadHttpRequest(requestPath string) (*http.Request, error) {
	request, err := os.ReadFile(requestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read request file: %w", err)
	}

	rawRequest := string(request)
	bufReader := bufio.NewReader(strings.NewReader(rawRequest))

	res, err := http.ReadRequest(bufReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read request file: %w", err)
	}

	return res, nil
}

func LoadHttpResponse(responsePath string) (*http.Response, error) {
	response, err := os.ReadFile(responsePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read response file: %w", err)
	}

	rawResponse := string(response)
	bufReader := bufio.NewReader(strings.NewReader(rawResponse))

	res, err := http.ReadResponse(bufReader, &http.Request{})
	if err != nil {
		return nil, fmt.Errorf("failed to read response file: %w", err)
	}

	return res, nil
}
