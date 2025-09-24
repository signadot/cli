package override

import (
	"fmt"
	"io"
	"time"

	"github.com/signadot/cli/internal/sdtab"
	"github.com/xeonx/timeago"
)

// TODO: Replace this with the spec from SDK
// Override represents a traffic override configuration
type Override struct {
	Name      string `json:"name"`
	Sandbox   string `json:"sandbox"`
	Target    string `json:"target"`
	Cluster   string `json:"cluster"`
	CreatedAt string `json:"createdAt"`
	Status    string `json:"status"`
	Detached  bool   `json:"detached"`
}

type overrideRow struct {
	Name    string `sdtab:"NAME"`
	Target  string `sdtab:"TARGET"`
	Status  string `sdtab:"STATUS"`
	Created string `sdtab:"CREATED"`
}

// printOverrideTable prints a table of overrides
func printOverrideTable(out io.Writer, overrides []*Override) error {
	t := sdtab.New[overrideRow](out)
	t.AddHeader()

	for _, override := range overrides {
		createdAt, err := time.Parse(time.RFC3339, override.CreatedAt)
		if err != nil {
			return err
		}

		t.AddRow(overrideRow{
			Name:    override.Name,
			Target:  fmt.Sprintf("sandbox=%s -> %s", override.Sandbox, override.Target),
			Status:  getOverrideStatus(override),
			Created: timeago.NoMax(timeago.English).Format(createdAt),
		})
	}

	return t.Flush()
}

// getOverrideStatus returns a human-readable status for the override
func getOverrideStatus(override *Override) string {
	if override.Detached {
		return "detached"
	}
	return "active"
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
