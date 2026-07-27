package logs

import (
	"context"
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
)

// followPollInterval is how often the sandbox path re-polls the (snapshot) logs
// endpoint when --follow is set. Native streaming is a fast-follow (ENG-1115).
const followPollInterval = 2 * time.Second

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
// exact message hasn't been printed yet — so distinct lines sharing the boundary
// timestamp are NOT dropped. Untimestamped lines can't be de-duplicated, so they
// are printed only on the first (snapshot) poll and skipped thereafter to avoid
// reprinting the whole window forever.
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

// sinceTime returns the next poll's sinceTime: the oldest per-container hwm (so
// no lagging container loses lines to the shared filter), or fallback when no
// timestamped line has been seen.
func (s *followState) sinceTime(fallback string) string {
	var oldest time.Time
	found := false
	for _, c := range s.byContainer {
		if !c.hasHWM {
			continue
		}
		if !found || c.hwm.Before(oldest) {
			oldest = c.hwm
			found = true
		}
	}
	if !found {
		return fallback
	}
	return oldest.UTC().Format(time.RFC3339Nano)
}

// isTransientFollowErr reports whether a follow-poll error is worth retrying
// (pod not running yet, restart, gateway/connection) rather than a selector
// mistake that won't self-heal. Heuristic on the error text pending typed
// errors (ENG-1115) — the readiness phrasings are matched explicitly so a
// "not ready yet" message isn't mistaken for a "<thing> not found" selector error.
func isTransientFollowErr(err error) bool {
	if err == nil {
		return false
	}
	m := err.Error()
	for _, s := range []string{
		"may not be ready", "no running pods", "not running",
		"502", "503", "504", "Gateway", "connection", "EOF", "timeout",
	} {
		if strings.Contains(m, s) {
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
