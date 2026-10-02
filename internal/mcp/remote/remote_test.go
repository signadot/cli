package remote

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/signadot/cli/internal/auth"
	"github.com/signadot/cli/internal/config"
	"github.com/zalando/go-keyring"
)

// After logging in again (e.g. to another org), Session must not keep using
// the session created with the previous credentials.
func TestSessionFollowsLogin(t *testing.T) {
	keyring.MockInit()
	t.Setenv("HOME", t.TempDir())

	var (
		mu      sync.Mutex
		apiKeys []string
	)
	server := mcp.NewServer(&mcp.Implementation{Name: "fake"}, nil)
	h := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		apiKeys = append(apiKeys, r.Header.Get("signadot-api-key"))
		mu.Unlock()
		h.ServeHTTP(w, r)
	}))
	defer srv.Close()

	r := NewRemoteManager(slog.New(slog.NewTextHandler(io.Discard, nil)),
		&config.MCP{API: &config.API{MCPURL: srv.URL}})
	r.remoteClient = mcp.NewClient(&mcp.Implementation{Name: "test"}, nil)
	defer r.Close()

	login := func(key, org string) {
		t.Helper()
		if err := auth.StoreExclusive(auth.NewKeyringStorage(), &auth.Auth{APIKey: key, OrgName: org}); err != nil {
			t.Fatal(err)
		}
	}

	login("key-a", "org-a")
	s1, err := r.Session()
	if err != nil {
		t.Fatal(err)
	}
	if s2, _ := r.Session(); s2 != s1 {
		t.Fatal("session recreated without a credentials change")
	}

	login("key-b", "org-b")
	s3, err := r.Session()
	if err != nil {
		t.Fatal(err)
	}
	if s3 == s1 {
		t.Fatal("session not recreated after login as another org")
	}

	mu.Lock()
	defer mu.Unlock()
	if last := apiKeys[len(apiKeys)-1]; last != "key-b" {
		t.Fatalf("last request used api key %q, want key-b", last)
	}
}

// The first metadata fetch attempt is signaled even when it fails, so the MCP
// server can start (with local tools) while offline.
func TestFirstFetchDoneOnFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	r := NewRemoteManager(slog.New(slog.NewTextHandler(io.Discard, nil)),
		&config.MCP{API: &config.API{MCPURL: srv.URL}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx, time.Hour)

	select {
	case <-r.FirstFetchDone():
	case <-time.After(5 * time.Second):
		t.Fatal("first fetch attempt not signaled")
	}
	if r.Meta() != nil {
		t.Fatal("unexpected metadata")
	}
}
