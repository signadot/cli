package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Meta represents the metadata exposed by the remote MCP server.
type Meta struct {
	Tools []*mcp.Tool `json:"tools"`
}

// updateMeta fetches and updates the cached metadata.
// It checks for changes and invokes the onChange callback if metadata has changed.
func (r *Remote) updateMeta(ctx context.Context) error {
	meta, err := r.fetchMeta(ctx)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if metadata has changed
	hasChanged := !reflect.DeepEqual(r.meta, meta)
	if !hasChanged {
		return nil
	}

	// Invoke callback if set (unlock mutex during callback to avoid deadlock)
	if r.onChange != nil {
		r.mu.Unlock()
		err = r.onChange(ctx, meta)
		r.mu.Lock()

		if err != nil {
			// Don't update the state if the callback fails
			return fmt.Errorf("failed to invoke onChange callback: %w", err)
		}
	}

	// Update metadata state
	r.meta = meta
	return nil
}

// fetchMeta fetches metadata from the remote server's /meta endpoint.
func (r *Remote) fetchMeta(ctx context.Context) (*Meta, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", r.cfg.MCPURL+"/meta", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var meta Meta
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}
	return &meta, nil
}
