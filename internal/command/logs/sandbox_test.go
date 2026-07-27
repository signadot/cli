package logs

import "testing"

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
}
