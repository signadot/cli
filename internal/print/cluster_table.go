package print

import (
	"io"

	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/models"
)

type clusterRow struct {
	Name    string `sdtab:"NAME"`
	Created string `sdtab:"CREATED"`
	Version string `sdtab:"OPERATOR VERSION"`
}

func ClusterTable(out io.Writer, clusters []*models.Cluster) error {
	t := sdtab.New[clusterRow](out)
	t.AddHeader()
	for _, cluster := range clusters {
		t.AddRow(clusterRow{
			Name:    cluster.Name,
			Created: cluster.CreatedAt,
			Version: cluster.OperatorVersion,
		})
	}
	return t.Flush()
}
