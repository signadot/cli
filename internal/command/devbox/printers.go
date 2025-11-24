package devbox

import (
	"io"
	"time"

	"github.com/signadot/cli/internal/sdtab"
	"github.com/xeonx/timeago"
)

// devboxRow represents a row in the devbox table output.
type devboxRow struct {
	Name       string `sdtab:"NAME"`
	Primary    string `sdtab:"PRIMARY"`
	HostName   string `sdtab:"HOSTNAME,trunc"`
	OS         string `sdtab:"OS"`
	MachineID  string `sdtab:"MACHINE ID,trunc"`
	LastActive string `sdtab:"LAST ACTIVE"`
	Status     string `sdtab:"STATUS"`
}

// TODO: Define the Devbox model type once API is implemented
// For now, using a placeholder interface
type DevboxModel interface{}

// printDevboxTable prints devboxes in a table format.
func printDevboxTable(out io.Writer, devboxes []DevboxModel) error {
	t := sdtab.New[devboxRow](out)
	t.AddHeader()

	// TODO: Iterate over actual devbox models once API is implemented
	// Example:
	// for _, db := range devboxes {
	//     lastActive := ""
	//     if db.Status != nil && db.Status.Session != nil {
	//         renewedAt, err := time.Parse(time.RFC3339, db.Status.Session.RenewedAt)
	//         if err == nil {
	//             lastActive = timeago.NoMax(timeago.English).Format(renewedAt)
	//         }
	//     }
	//
	//     primary := "no"
	//     if db.Machine != nil && db.Machine.Primary {
	//         primary = "yes"
	//     }
	//
	//     status := "inactive"
	//     if db.Status != nil && db.Status.Session != nil {
	//         validUntil, err := time.Parse(time.RFC3339, db.Status.Session.ValidUntil)
	//         if err == nil && validUntil.After(time.Now()) {
	//             status = "active"
	//         }
	//     }
	//
	//     t.AddRow(devboxRow{
	//         Name:       db.Machine.Name,
	//         Primary:    primary,
	//         HostName:   db.Machine.Meta.HostName,
	//         OS:         db.Machine.Meta.OS,
	//         MachineID:  db.Machine.Meta.LocalMachineID,
	//         LastActive: lastActive,
	//         Status:     status,
	//     })
	// }

	// Placeholder row for testing
	t.AddRow(devboxRow{
		Name:       "example-devbox",
		Primary:    "yes",
		HostName:   "localhost",
		OS:         "linux",
		MachineID:  "abc123",
		LastActive: timeago.NoMax(timeago.English).Format(time.Now()),
		Status:     "inactive",
	})

	return t.Flush()
}
