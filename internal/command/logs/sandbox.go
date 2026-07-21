package logs

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
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

// followSandboxLogs tails a sandbox source by polling the snapshot endpoint. It
// advances sinceTime to the oldest per-container high-water mark and de-dupes
// each container against its last-printed timestamp, so multiple containers
// (which share a single sinceTime per request) don't lose or repeat lines. The
// first poll surfaces selector errors (unknown workload/container/etc.);
// afterwards transient errors (pod not running yet, restarts) are retried.
func followSandboxLogs(ctx context.Context, out, errW io.Writer, cfg *config.Logs, initialSince string) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	lastByContainer := map[string]time.Time{}
	since := initialSince
	first := true
	var lastErrMsg string

	for {
		resp, err := fetchSandboxLogs(ctx, cfg, since)
		if err != nil {
			if ctx.Err() != nil {
				return nil // interrupted
			}
			if first {
				return err // validate selectors up front
			}
			if msg := err.Error(); msg != lastErrMsg {
				fmt.Fprintf(errW, "waiting for logs: %s\n", msg)
				lastErrMsg = msg
			}
		} else {
			lastErrMsg = ""
			multi := len(resp) > 1
			for _, cl := range resp {
				last := lastByContainer[cl.Container]
				for _, item := range cl.Logs {
					t, perr := time.Parse(time.RFC3339, item.Time)
					if perr == nil && !first && !t.After(last) {
						continue // already printed (boundary dedup)
					}
					printLogLine(out, cl.Container, item.Message, multi)
					if perr == nil && t.After(lastByContainer[cl.Container]) {
						lastByContainer[cl.Container] = t
					}
				}
			}
			since = oldestSince(lastByContainer, since)
			first = false
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(followPollInterval):
		}
	}
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

// oldestSince returns the RFC3339 timestamp to use for the next poll: the oldest
// per-container high-water mark (so no container's newer-than-its-own-last lines
// are skipped by the shared sinceTime). Falls back to the current value when no
// lines have been seen yet.
func oldestSince(lastByContainer map[string]time.Time, current string) string {
	var oldest time.Time
	for _, t := range lastByContainer {
		if oldest.IsZero() || t.Before(oldest) {
			oldest = t
		}
	}
	if oldest.IsZero() {
		return current
	}
	return oldest.UTC().Format(time.RFC3339Nano)
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
