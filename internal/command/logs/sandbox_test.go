package logs

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/signadot/go-sdk/transport"
)

// step is one observe() call in a scripted follow sequence.
type step struct {
	container string
	time      string
	message   string
	first     bool
	wantPrint bool
}

func runObserveSteps(t *testing.T, name string, steps []step) {
	t.Helper()
	st := newFollowState()
	for i, s := range steps {
		if got := st.observe(s.container, s.time, s.message, s.first); got != s.wantPrint {
			t.Errorf("%s step %d observe(%q,%q,%q,first=%v)=%v, want %v",
				name, i, s.container, s.time, s.message, s.first, got, s.wantPrint)
		}
	}
}

func TestFollowStateObserve(t *testing.T) {
	// Duplicate timestamp across a poll boundary: the re-sent line at the
	// high-water timestamp is suppressed, but a *distinct* line sharing that
	// exact timestamp is still printed (regression guard for the naive
	// "<= last" filter that would drop it).
	runObserveSteps(t, "boundary-same-timestamp", []step{
		{"app", "2026-01-01T00:00:02Z", "a", true, true},   // poll1: new
		{"app", "2026-01-01T00:00:02Z", "a", false, false}, // poll2: exact dup at boundary
		{"app", "2026-01-01T00:00:02Z", "b", false, true},  // poll2: distinct line, same ts -> keep
		{"app", "2026-01-01T00:00:03Z", "c", false, true},  // poll2: newer
	})

	// A line older than the high-water mark was already printed earlier.
	runObserveSteps(t, "older-than-hwm", []step{
		{"app", "2026-01-01T00:00:05Z", "x", true, true},
		{"app", "2026-01-01T00:00:03Z", "y", false, false},
	})

	// Untimestamped lines can't be de-duplicated: print on the first poll,
	// skip afterwards so the window isn't reprinted forever.
	runObserveSteps(t, "empty-timestamp", []step{
		{"svc", "", "m", true, true},
		{"svc", "", "m", false, false},
		{"svc", "", "n", false, false},
	})

	// Two containers advance independently and don't suppress each other.
	runObserveSteps(t, "independent-containers", []step{
		{"app", "2026-01-01T00:00:02Z", "a", true, true},
		{"sidecar", "2026-01-01T00:00:02Z", "a", true, true}, // same ts+msg, different container
		{"sidecar", "2026-01-01T00:00:02Z", "a", false, false},
	})
}

func TestFollowStateSinceTime(t *testing.T) {
	// No lines seen yet -> fall back to the caller's value.
	if got := newFollowState().sinceTime("FALLBACK"); got != "FALLBACK" {
		t.Errorf("sinceTime with no data = %q, want FALLBACK", got)
	}

	// Lagging container: use the OLDEST per-container high-water mark so the
	// shared sinceTime doesn't skip the lagging container's newer lines.
	st := newFollowState()
	st.observe("fast", "2026-01-01T00:00:05Z", "a", true)
	st.observe("slow", "2026-01-01T00:00:02Z", "b", true)
	if got, want := st.sinceTime("FALLBACK"), "2026-01-01T00:00:02Z"; got != want {
		t.Errorf("sinceTime = %q, want %q (oldest hwm)", got, want)
	}

	// A container with only untimestamped lines contributes no hwm.
	st2 := newFollowState()
	st2.observe("notime", "", "x", true)
	if got := st2.sinceTime("FB"); got != "FB" {
		t.Errorf("sinceTime with only untimestamped = %q, want FB", got)
	}

	// Rewind clamp: a container that logged once and went quiet sits far behind
	// the newest mark; sinceTime must not rewind past newest-grace, else every
	// poll re-requests that stale container's whole window forever.
	st3 := newFollowState()
	st3.observe("live", "2026-01-01T00:01:00Z", "a", true)
	st3.observe("stopped", "2026-01-01T00:00:00Z", "b", true) // 60s behind, grace is smaller
	newest := time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC)
	want := newest.Add(-followRewindGrace).UTC().Format(time.RFC3339Nano)
	if got := st3.sinceTime("FB"); got != want {
		t.Errorf("sinceTime clamp = %q, want %q (newest-grace)", got, want)
	}
}

// wrappedAPIErr mimics what the go-sdk FixAPIErrors middleware returns: a
// *transport.APIError (clean server message + HTTP status code) wrapped behind
// the HTTP status text, e.g. "400 Bad Request: <message>". The wrapping is what
// makes err.Error() dangerous to match on and why the guard uses errors.As.
func wrappedAPIErr(code int, message string) error {
	apiErr := &transport.APIError{}
	apiErr.Code = int64(code)
	apiErr.ErrorResponse.Error = message
	return fmt.Errorf("%d %s: %w", code, "Status", apiErr)
}

func TestIsTransientFollowErr(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"5xx gateway", wrappedAPIErr(502, "can't fetch logs from cluster"), true},
		{"503 unavailable", wrappedAPIErr(503, "can't fetch pod from cluster"), true},
		{"4xx readiness (not ready yet)", wrappedAPIErr(400, "no running pods found for the forked workload; the sandbox may not be ready yet"), true},
		// Regression (Scott): a fork/container named "connection-pool" or
		// "timeout-worker" puts "connection"/"timeout" in the message, but it's
		// a plain selector error -> must fail fast, not retry forever.
		{"4xx selector with connection-pool name", wrappedAPIErr(400, `container "connection-pool" not found; containers: app, istio-proxy`), false},
		{"4xx selector with timeout-worker name", wrappedAPIErr(400, `workload "timeout-worker" not found in sandbox "s"; loggable workloads: app`), false},
		{"transport failure (no response)", errors.New("dial tcp 10.0.0.1:443: connect: connection refused"), true},
		{"transport EOF", errors.New("unexpected EOF"), true},
		{"unrelated non-API error", errors.New("something else entirely"), false},
	}
	for _, c := range cases {
		if got := isTransientFollowErr(c.err); got != c.want {
			t.Errorf("%s: isTransientFollowErr(%v) = %v, want %v", c.name, c.err, got, c.want)
		}
	}
}
