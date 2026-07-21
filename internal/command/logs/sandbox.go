package logs

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/go-sdk/client/sandboxes"
)

// showSandboxLogs fetches logs for a sandbox workload (fork) or resource via the
// sandbox-scoped logs endpoint and prints them per-container.
func showSandboxLogs(ctx context.Context, out io.Writer, cfg *config.Logs) error {
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

	sinceTime, err := resolveSinceTime(cfg.Since, cfg.SinceTime)
	if err != nil {
		return err
	}

	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

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
		return err
	}

	switch cfg.OutputFormat {
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	case config.OutputFormatDefault:
		multi := len(resp.Payload) > 1
		for _, cl := range resp.Payload {
			for _, item := range cl.Logs {
				if multi {
					fmt.Fprintf(out, "[%s] %s\n", cl.Container, item.Message)
				} else {
					fmt.Fprintln(out, item.Message)
				}
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
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
