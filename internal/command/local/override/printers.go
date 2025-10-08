package override

import (
	"fmt"
	"io"
	"net"

	"github.com/signadot/cli/internal/builder"
	"github.com/signadot/cli/internal/sdtab"
)

type sandboxWithForward struct {
	Sandbox  string
	Forwards []*builder.DetailedOverrideMiddleware
}

type overrideRow struct {
	Name    string `sdtab:"NAME"`
	Target  string `sdtab:"TARGET"`
	ToLocal string `sdtab:"TO"`
	Status  string `sdtab:"STATUS"`
}

func isOverrideAttachedRunning(forward *builder.DetailedOverrideMiddleware) bool {
	// Ping the log forward to see if it is running
	conn, err := net.Dial("tcp", forward.LogForward.ToLocal)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

// printOverrideTable prints a table of overrides
func printOverrideTable(out io.Writer, sandboxes []*sandboxWithForward) error {
	t := sdtab.New[overrideRow](out)
	t.AddHeader()

	for _, override := range sandboxes {
		for _, forward := range override.Forwards {

			var status string

			switch {
			case forward.LogForward != nil:
				if isOverrideAttachedRunning(forward) {
					status = "attached"
				} else {
					status = "stopped"
				}
			default:
				status = "detached"
			}

			t.AddRow(overrideRow{
				Name:    forward.Forward.Name,
				Target:  fmt.Sprintf("sandbox=%s", override.Sandbox),
				ToLocal: forward.Forward.ToLocal,
				Status:  status,
			})
		}
	}

	return t.Flush()
}

// printOverrideStatus prints the status of an override operation
func printOverrideStatus(out io.Writer, message string, success bool) {
	if success {
		fmt.Fprintf(out, "✓ %s\n", message)
	} else {
		fmt.Fprintf(out, "✗ %s\n", message)
	}
}

// printOverrideProgress prints progress messages during override operations
func printOverrideProgress(out io.Writer, message string) {
	fmt.Fprintf(out, "→ %s\n", message)
}
