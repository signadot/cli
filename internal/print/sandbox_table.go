package print

import (
	"io"

	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/models"
)

type sandboxRow struct {
	Name        string `sdtab:"NAME"`
	Description string `sdtab:"DESCRIPTION,trunc"`
	Cluster     string `sdtab:"CLUSTER"`
	Created     string `sdtab:"CREATED"`
}

func SandboxTable(out io.Writer, sbs []*models.SandboxInfo) error {
	t := sdtab.New[sandboxRow](out)
	t.AddHeader()
	for _, sbinfo := range sbs {
		t.AddRow(sandboxRow{
			Name:        sbinfo.Name,
			Description: sbinfo.Description,
			Cluster:     sbinfo.ClusterName,
			Created:     sbinfo.CreatedAt,
		})
	}
	return t.Flush()
}
