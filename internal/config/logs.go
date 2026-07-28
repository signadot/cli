package config

import (
	"github.com/spf13/cobra"
)

type Logs struct {
	*API

	// Job log source (streamed).
	Job    string
	Stream string

	// Sandbox log source (snapshot). Workload and Resource are mutually
	// exclusive; Step only applies to Resource.
	Sandbox   string
	Workload  string
	Resource  string
	Step      string
	Container string

	// Selectors shared where applicable.
	TailLines uint
	Since     string
	SinceTime string
}

func (c *Logs) AddFlags(cmd *cobra.Command) {
	// Job path (existing, backward-compatible).
	cmd.Flags().StringVarP(&c.Job, "job", "j", "", "job name whose log lines will be displayed")
	cmd.Flags().StringVarP(&c.Stream, "stream", "s", "stdout", "stream to display for a job (stdout or stderr); job-only")
	cmd.Flags().UintVarP(&c.TailLines, "tail", "t", 0, "lines of recent log to display, defaults to 0 (all)")

	// Sandbox path.
	cmd.Flags().StringVar(&c.Sandbox, "sandbox", "", "sandbox name whose workload/resource logs will be displayed")
	cmd.Flags().StringVar(&c.Workload, "workload", "", "sandboxed workload (fork) name from the sandbox spec")
	cmd.Flags().StringVar(&c.Resource, "resource", "", "sandbox resource name from the sandbox spec")
	cmd.Flags().StringVar(&c.Step, "step", "", "resource plugin step name (only with --resource)")
	cmd.Flags().StringVarP(&c.Container, "container", "c", "", "container name to display; defaults to all containers")
	cmd.Flags().StringVar(&c.Since, "since", "", "only display logs newer than a relative duration, e.g. 10m, 1h, 2h30m")
	cmd.Flags().StringVar(&c.SinceTime, "since-time", "", "only display logs after an RFC3339 timestamp")

	// --job and --sandbox are different log sources. Reject cross-source flags
	// rather than silently ignoring them:
	//   - job-only flags (--stream, --tail) can't combine with --sandbox
	//     (the sandbox path doesn't read them; server-side tail is a fast-follow),
	//   - sandbox-only selectors can't combine with --job.
	cmd.MarkFlagsMutuallyExclusive("job", "sandbox")
	cmd.MarkFlagsMutuallyExclusive("stream", "sandbox")
	cmd.MarkFlagsMutuallyExclusive("tail", "sandbox")
	for _, sandboxOnly := range []string{"workload", "resource", "step", "container", "since", "since-time"} {
		cmd.MarkFlagsMutuallyExclusive("job", sandboxOnly)
	}
}
