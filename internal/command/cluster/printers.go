package cluster

import (
	"io"

	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/models"
)

type clusterRow struct {
	Name     string `sdtab:"NAME"`
	Created  string `sdtab:"CREATED"`
	Version  string `sdtab:"OPERATOR VERSION"`
	LastSync string `sdtab:"LAST SYNC"`
}

func printClusterTable(out io.Writer, clusters []*models.Cluster) error {
	t := sdtab.New[clusterRow](out)
	t.AddHeader()
	for _, cluster := range clusters {
		lastSync := "-"
		if cluster.SyncStatus != nil {
			syncStat := cluster.SyncStatus
			if syncStat.LastSync != nil {
				if syncStat.LastSync.Time != "" {
					lastSync = syncStat.LastSync.Time
				}
			}
		}
		t.AddRow(clusterRow{
			Name:     cluster.Name,
			Created:  cluster.CreatedAt,
			Version:  cluster.Operator.Version,
			LastSync: lastSync,
		})
	}
	return t.Flush()
}
