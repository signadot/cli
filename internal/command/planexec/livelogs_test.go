package planexec

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"
)

func sseMsg(w io.Writer, msg, cursor string) {
	d, _ := json.Marshal(map[string]string{"message": msg, "cursor": cursor})
	fmt.Fprintf(w, "event: message\ndata: %s\n\n", d)
}

// The live stream is resumed from the last cursor after a 404 before
// dispatch and after the stream is cut (e.g. the server's idle timeout).
func TestFollowLiveLogsResumes(t *testing.T) {
	keyring.MockInit()
	t.Setenv("HOME", t.TempDir())
	defer func(d time.Duration) { liveLogRetryDelay = d }(liveLogRetryDelay)
	liveLogRetryDelay = time.Millisecond

	var (
		mu      sync.Mutex
		streams int
		cursors []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/logs/live") {
			// execution status: still running
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"id":"ex1","status":{"phase":"running"}}`)
			return
		}
		mu.Lock()
		streams++
		n := streams
		cursors = append(cursors, r.URL.Query().Get("cursor"))
		mu.Unlock()
		switch n {
		case 1:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, `{"error":"execution not yet dispatched to a runner"}`)
		case 2:
			w.Header().Set("Content-Type", "text/event-stream")
			sseMsg(w, "line1\n", "c1")
			io.WriteString(w, "event: error\ndata: log stream timed out: no data received\n\n")
		default:
			w.Header().Set("Content-Type", "text/event-stream")
			sseMsg(w, "line2\n", "c2")
			io.WriteString(w, "event: signal\ndata: EOF\n\n")
		}
	}))
	defer srv.Close()
	viper.Set("api_key", "key")
	viper.Set("org", "org")
	viper.Set("api_url", srv.URL)
	defer viper.Reset()

	var out strings.Builder
	err := FollowLiveLogs(context.Background(), initAPI(t), "ex1", LiveLogOptions{},
		func(r io.Reader) (string, error) { return print.ParseSSEStream(r, &out) })
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != "line1\nline2\n" {
		t.Errorf("output = %q", out.String())
	}
	if want := []string{"", "", "c1"}; fmt.Sprint(cursors) != fmt.Sprint(want) {
		t.Errorf("cursors = %q, want %q", cursors, want)
	}
}

// Once the execution has finished, a 404 ends following (with the error).
func TestFollowLiveLogsStopsWhenFinished(t *testing.T) {
	keyring.MockInit()
	t.Setenv("HOME", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/logs/live") {
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, `{"error":"live plan logs are unavailable"}`)
			return
		}
		io.WriteString(w, `{"id":"ex1","status":{"phase":"completed"}}`)
	}))
	defer srv.Close()
	viper.Set("api_key", "key")
	viper.Set("org", "org")
	viper.Set("api_url", srv.URL)
	defer viper.Reset()

	err := FollowLiveLogs(context.Background(), initAPI(t), "ex1", LiveLogOptions{},
		func(r io.Reader) (string, error) { return print.ParseSSEStream(r, io.Discard) })
	if err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("got %v, want live logs unavailable error", err)
	}
}

func initAPI(t *testing.T) *config.API {
	t.Helper()
	cfg := &config.API{}
	if err := cfg.InitAPIConfig(); err != nil {
		t.Fatal(err)
	}
	return cfg
}
