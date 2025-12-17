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
	ID        string `sdtab:"ID"`
	Name      string `sdtab:"NAME,trunc"`
	OS        string `sdtab:"OS"`
	MachineID string `sdtab:"MACHINE ID,trunc"`
	Status    string `sdtab:"STATUS"`
}

// printDevboxTable prints devboxes in a table format.
func printDevboxTable(out io.Writer, devboxes []*models.Devbox, currentDevboxID string) error {
	t := sdtab.New[devboxRow](out)
	t.AddHeader()

	if currentDevboxID != "" {
		// ensure stored devbox is first
		for i, b := range devboxes {
			if b.ID != currentDevboxID {
				continue
			}
			if i == 0 {
				break
			}
			hdb := devboxes[0]
			devboxes[0], devboxes[i] = b, hdb
			break
		}
	}

	for _, db := range devboxes {
		status := "inactive"
		if db.Status != nil && db.Status.Session != nil && db.Status.Session.ValidUntil != "" {
			validUntil, err := time.Parse(time.RFC3339, db.Status.Session.ValidUntil)
			if err == nil {
				now := time.Now()
				if validUntil.After(now) {
					status = "active"
				} else {
					// Session has expired, show how long ago
					duration := now.Sub(validUntil)
					timeAgo := timeago.NoMax(timeago.English).FormatRelativeDuration(duration)
					status = "expired " + timeAgo
				}
			}
		}

		// Mark current devbox as "default" in status column
		isDefault := currentDevboxID != "" && db.ID == currentDevboxID
		if isDefault {
			status = status + " (*)"
		}

		name := db.Metadata["name"]
		os := db.Metadata["goos"]
		machineID := db.Metadata["machine-id"]
		// Truncate machine ID to 12 characters with "..." suffix
		machineID = sdtab.Truncate(machineID, 12)
		if db.Status.Session != nil {
			ses := db.Status.Session
			_ = ses
		}

		t.AddRow(devboxRow{
			ID:        db.ID,
			Name:      name,
			OS:        os,
			MachineID: machineID,
			Status:    status,
		})
	}

	return t.Flush()
}
