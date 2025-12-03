package devbox

import (
	"io"
	"time"

	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/models"
	"github.com/xeonx/timeago"
)

// devboxRow represents a row in the devbox table output.
type devboxRow struct {
	ID         string `sdtab:"ID"`
	Name       string `sdtab:"NAME,trunc"`
	OS         string `sdtab:"OS"`
	MachineID  string `sdtab:"MACHINE ID,trunc"`
	LastActive string `sdtab:"LAST ACTIVE"`
	Status     string `sdtab:"STATUS"`
}

// printDevboxTable prints devboxes in a table format.
func printDevboxTable(out io.Writer, devboxes []*models.Devbox) error {
	t := sdtab.New[devboxRow](out)
	t.AddHeader()

	for _, db := range devboxes {
		lastActive := ""
		if db.Status != nil && db.Status.Session != nil && db.Status.Session.RenewedAt != "" {
			renewedAt, err := time.Parse(time.RFC3339, db.Status.Session.RenewedAt)
			if err == nil {
				lastActive = timeago.NoMax(timeago.English).Format(renewedAt)
			}
		}

		status := "inactive"
		if db.Status != nil && db.Status.Session != nil && db.Status.Session.ValidUntil != "" {
			validUntil, err := time.Parse(time.RFC3339, db.Status.Session.ValidUntil)
			if err == nil && validUntil.After(time.Now()) {
				status = "active"
			}
		}

		name := db.IDMeta["name"]
		os := db.Labels["goos"]
		machineID := db.IDMeta["machine-id"]
		// Truncate machine ID to 12 characters with "..." suffix
		machineID = sdtab.Truncate(machineID, 12)
		if db.Status.Session != nil {
			ses := db.Status.Session
			_ = ses
		}

		t.AddRow(devboxRow{
			ID:         db.ID,
			Name:       name,
			OS:         os,
			MachineID:  machineID,
			LastActive: lastActive,
			Status:     status,
		})
	}

	return t.Flush()
}
