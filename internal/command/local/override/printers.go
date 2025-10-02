package override

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/models"
)

type sandboxWithForward struct {
	Sandbox  string
	Forwards []*models.SandboxesForward
}

type overrideRow struct {
	Name    string `sdtab:"NAME"`
	Target  string `sdtab:"TARGET"`
	ToLocal string `sdtab:"TO"`
}

// printOverrideTable prints a table of overrides
func printOverrideTable(out io.Writer, sandboxes []*sandboxWithForward) error {
	t := sdtab.New[overrideRow](out)
	t.AddHeader()

	for _, override := range sandboxes {
		for _, forward := range override.Forwards {
			t.AddRow(overrideRow{
				Name:    forward.Name,
				Target:  fmt.Sprintf("sandbox=%s", override.Sandbox),
				ToLocal: forward.ToLocal,
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
