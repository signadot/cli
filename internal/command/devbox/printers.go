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
	HostName   string `sdtab:"HOSTNAME,trunc"`
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

		hostName := db.Labels["host"]
		os := db.Labels["goos"]
		machineID := db.IDMeta["machine-id"]
		if db.Status.Session != nil {
			ses := db.Status.Session
			_ = ses
		}

		t.AddRow(devboxRow{
			ID:         db.ID,
			HostName:   hostName,
			OS:         os,
			MachineID:  machineID,
			LastActive: lastActive,
			Status:     status,
		})
	}

	return t.Flush()
}
