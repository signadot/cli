package logs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
	"github.com/signadot/go-sdk/transport"
)

// followPollInterval is how often the sandbox path re-polls the (snapshot) logs
// endpoint when --follow is set. Native streaming is a fast-follow (ENG-1115).
const followPollInterval = 2 * time.Second

// followRewindGrace bounds how far back the shared sinceTime may rewind for the
// next poll (see followState.sinceTime). A few poll intervals: enough that a
// still-active container that's merely lagging keeps its lines, without pinning
// the whole request window to a container that logged once and went quiet.
const followRewindGrace = 3 * followPollInterval

// serverTailCap mirrors the control plane's per-container logReqTailLines cap
// (sandboxes/internal/control/logs/pods/fetch.go). A poll that returns exactly
// this many lines for a container was almost certainly truncated, so --follow
// warns rather than silently dropping the older lines.
const serverTailCap = 1000

// showSandboxLogs fetches logs for a sandbox workload (fork) or resource via the
// sandbox-scoped logs endpoint and prints them per-container. With --follow it
// polls, advancing sinceTime and de-duplicating per container.
func showSandboxLogs(ctx context.Context, out, errW io.Writer, cfg *config.Logs) error {
	if cfg.Workload == "" && cfg.Resource == "" {
		return fmt.Errorf("must specify --workload or --resource with --sandbox")
	}
	if cfg.Workload != "" && cfg.Resource != "" {
		return fmt.Errorf("--workload and --resource are mutually exclusive")
	}
	if cfg.Step != "" && cfg.Resource == "" {
		return fmt.Errorf("--step is only valid with --resource")
	}
	if cfg.Since != "" && cfg.SinceTime != "" {
		return fmt.Errorf("--since and --since-time are mutually exclusive")
	}
	if cfg.Follow && cfg.OutputFormat != config.OutputFormatDefault {
		return fmt.Errorf("--follow cannot be combined with -o %s", cfg.OutputFormat)
	}

	sinceTime, err := resolveSinceTime(cfg.Since, cfg.SinceTime)
	if err != nil {
		return err
	}

	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	if cfg.Follow {
		return followSandboxLogs(ctx, out, errW, cfg, sinceTime)
	}

	resp, err := fetchSandboxLogs(ctx, cfg, sinceTime)
	if err != nil {
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp)
	case config.OutputFormatDefault:
		multi := len(resp) > 1
		for _, cl := range resp {
			for _, item := range cl.Logs {
				printLogLine(out, cl.Container, item.Message, multi)
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}

// followSandboxLogs tails a sandbox source by polling the snapshot endpoint,
// advancing sinceTime and de-duplicating via followState (see its docs for the
// per-container high-water-mark + boundary-set scheme). On the first poll a
// selector error (unknown workload/container/etc.) fails fast; transient errors
// (pod not running yet, restarts, gateway/connection) are reported once and
// retried. Stops cleanly on SIGINT/SIGTERM/SIGHUP.
func followSandboxLogs(ctx context.Context, out, errW io.Writer, cfg *config.Logs, initialSince string) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer stop()

	st := newFollowState()
	since := initialSince
	first := true
	multi := false
	var lastErrMsg string
	capped := map[string]bool{} // containers currently hitting the server tail cap

	for {
		resp, err := fetchSandboxLogs(ctx, cfg, since)
		if err != nil {
			if ctx.Err() != nil {
				return nil // interrupted
			}
			if first && !isTransientFollowErr(err) {
				return err // fail fast on a selector mistake; it won't self-heal
			}
			if msg := err.Error(); msg != lastErrMsg {
				fmt.Fprintf(errW, "waiting for logs: %s\n", msg)
				lastErrMsg = msg
			}
		} else {
			lastErrMsg = ""
			// Latch the multi-container prefix once we've seen more than one
			// container, so a container that starts logging later doesn't flip
			// earlier lines between prefixed and unprefixed forms.
			if len(resp) > 1 {
				multi = true
			}
			for _, cl := range resp {
				for _, item := range cl.Logs {
					if st.observe(cl.Container, item.Time, item.Message, first) {
						printLogLine(out, cl.Container, item.Message, multi)
					}
				}
				// The server caps each container at serverTailCap lines per
				// fetch; hitting it means older lines in this interval were
				// dropped. Warn once per burst (and again if it recurs after
				// recovering) rather than silently losing lines under load.
				if hit := len(cl.Logs) >= serverTailCap; hit != capped[cl.Container] {
					if hit {
						fmt.Fprintf(errW, "warning: %q is logging faster than --follow can poll; showing only the most recent %d lines per %s (older lines dropped)\n",
							cl.Container, serverTailCap, followPollInterval)
					}
					capped[cl.Container] = hit
				}
			}
			since = st.sinceTime(since)
			first = false
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(followPollInterval):
		}
	}
}

// followState de-duplicates streamed log lines across polls. Because every
// container in a request shares a single sinceTime, we can't advance a global
// watermark without losing lines from a lagging container; instead each
// container keeps its own high-water mark (hwm) plus the set of messages seen at
// exactly that timestamp (boundary). A re-fetched line is printed iff it is
// strictly newer than the container's hwm, or shares the hwm timestamp but its
// exact message hasn't been printed yet. This keeps the client from dropping a
// distinct line that shares the boundary timestamp with one already printed —
// but only among the lines the server re-sends: the container that *defines* the
// shared sinceTime won't receive such a line at all, because the server filters
// on a strict After(sinceTime). Untimestamped lines can't be de-duplicated, so
// they are printed only on the first (snapshot) poll and skipped thereafter to
// avoid reprinting the whole window forever.
type followState struct {
	byContainer map[string]*containerCursor
}

type containerCursor struct {
	hwm      time.Time
	hasHWM   bool
	boundary map[string]struct{}
}

func newFollowState() *followState {
	return &followState{byContainer: map[string]*containerCursor{}}
}

// observe records a line and reports whether it should be printed now. first is
// true on the initial (snapshot) poll.
func (s *followState) observe(container, tstr, message string, first bool) bool {
	c := s.byContainer[container]
	if c == nil {
		c = &containerCursor{boundary: map[string]struct{}{}}
		s.byContainer[container] = c
	}

	t, err := time.Parse(time.RFC3339, tstr)
	if tstr == "" || err != nil {
		// No usable timestamp: can't de-dup across polls, so only emit on the
		// first poll. (This endpoint always timestamps; this is a safety net.)
		return first
	}

	switch {
	case !c.hasHWM || t.After(c.hwm):
		c.hwm = t
		c.hasHWM = true
		c.boundary = map[string]struct{}{message: {}}
		return true
	case t.Equal(c.hwm):
		if _, seen := c.boundary[message]; seen {
			return false
		}
		c.boundary[message] = struct{}{}
		return true
	default: // older than hwm — already printed in an earlier poll
		return false
	}
}

// sinceTime returns the next poll's sinceTime. All containers share one
// sinceTime, so it rewinds to the oldest per-container hwm to avoid dropping a
// lagging container's lines — but no further back than followRewindGrace behind
// the newest hwm. Without that clamp a container that logs once at startup and
// goes quiet (a startup-only sidecar such as istio-proxy, or a completed
// resource create-step pod) pins sinceTime at its stale mark forever, so every
// poll re-requests the whole capped window from it through the cluster tunnel.
// Containers in a pod share a clock, so a genuinely lagging *active* container
// won't be more than a few poll intervals behind the newest line. Returns
// fallback when no timestamped line has been seen.
func (s *followState) sinceTime(fallback string) string {
	var oldest, newest time.Time
	found := false
	for _, c := range s.byContainer {
		if !c.hasHWM {
			continue
		}
		if !found {
			oldest, newest, found = c.hwm, c.hwm, true
			continue
		}
		if c.hwm.Before(oldest) {
			oldest = c.hwm
		}
		if c.hwm.After(newest) {
			newest = c.hwm
		}
	}
	if !found {
		return fallback
	}
	if floor := newest.Add(-followRewindGrace); oldest.Before(floor) {
		oldest = floor
	}
	return oldest.UTC().Format(time.RFC3339Nano)
}

// isTransientFollowErr reports whether a follow-poll error is worth retrying
// (pod not running yet, restart, gateway/tunnel hiccup) rather than a selector
// mistake that won't self-heal.
//
// It decides on the go-sdk's *transport.APIError (the FixAPIErrors middleware
// wraps every 4xx/5xx in one), never on err.Error(): the formatted string
// prefixes the HTTP status text and splices in the selector the user typed plus
// the list of valid names, so a fork or container legitimately named
// "connection-pool" or "timeout-worker" would make a genuine typo match a
// substring and retry forever. A 5xx is a gateway/tunnel/k8s hiccup (retry); a
// 4xx is a client mistake and only the readiness cases — which the server also
// returns as 4xx ("...may not be ready yet" / "no running pods found...") —
// are transient, matched against the server's own message field alone. A
// non-API error is a transport failure before the server answered, so retry.
func isTransientFollowErr(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *transport.APIError
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.Code >= 500:
			return true
		case apiErr.Code >= 400:
			// ErrorResponse.Error is the clean server message — no status
			// prefix, no request id, no echoed selector list.
			return isReadinessMsg(apiErr.ErrorResponse.Error)
		default:
			return false
		}
	}
	// Not an API response: the request failed at the transport layer
	// (connection reset, EOF, timeout) before the server replied — retry.
	return isTransportErr(err.Error())
}

// isReadinessMsg matches the server's "sandbox not ready yet" messages, the
// only 4xx cases worth retrying under --follow.
func isReadinessMsg(msg string) bool {
	return strings.Contains(msg, "may not be ready") ||
		strings.Contains(msg, "no running pods")
}

// isTransportErr matches connection-level failures (no HTTP response received).
func isTransportErr(msg string) bool {
	for _, s := range []string{"connection", "EOF", "timeout", "no such host", "reset by peer"} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

// fetchSandboxLogs performs a single GetSandboxLogs call.
func fetchSandboxLogs(ctx context.Context, cfg *config.Logs, sinceTime string) ([]*models.LogsContainerLogs, error) {
	params := sandboxes.NewGetSandboxLogsParams().
		WithContext(ctx).
		WithOrgName(cfg.Org).
		WithSandboxName(cfg.Sandbox)
	params.Workload = optString(cfg.Workload)
	params.Resource = optString(cfg.Resource)
	params.Step = optString(cfg.Step)
	params.Container = optString(cfg.Container)
	params.SinceTime = optString(sinceTime)

	resp, err := cfg.Client.Sandboxes.GetSandboxLogs(params, nil)
	if err != nil {
		return nil, err
	}
	return resp.Payload, nil
}

func printLogLine(out io.Writer, container, message string, multi bool) {
	if multi {
		fmt.Fprintf(out, "[%s] %s\n", container, message)
	} else {
		fmt.Fprintln(out, message)
	}
}

// resolveSinceTime converts the CLI's --since (relative duration) or
// --since-time (absolute RFC3339) into the RFC3339 sinceTime the API expects.
func resolveSinceTime(since, sinceTime string) (string, error) {
	switch {
	case since != "":
		d, err := time.ParseDuration(since)
		if err != nil {
			return "", fmt.Errorf("invalid --since duration %q: %w", since, err)
		}
		return time.Now().Add(-d).UTC().Format(time.RFC3339), nil
	case sinceTime != "":
		if _, err := time.Parse(time.RFC3339, sinceTime); err != nil {
			return "", fmt.Errorf("invalid --since-time %q (want RFC3339): %w", sinceTime, err)
		}
		return sinceTime, nil
	default:
		return "", nil
	}
}

func optString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
