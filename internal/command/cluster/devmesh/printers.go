package devmesh

import (
	"fmt"
	"io"
	"strings"

	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/models"
)

type devMeshAnalysisRow struct {
	Kind      string `sdtab:"KIND"`
	Namespace string `sdtab:"NAMESPACE"`
	Name      string `sdtab:"NAME"`
	Updated   string `sdtab:"UPDATED"`
	Status    string `sdtab:"STATUS"`
	Reason    string `sdtab:"REASON"`
}

func printDevMeshAnalysisTable(out io.Writer, workloads []*models.ClustersDevMeshEnabledWorkload) error {
	t := sdtab.New[devMeshAnalysisRow](out)
	t.AddHeader()

	for _, w := range workloads {
		up, reason := getUpdatedAndReason(w)
		t.AddRow(devMeshAnalysisRow{
			Kind:      *w.Workload.Kind,
			Namespace: *w.Workload.Namespace,
			Name:      *w.Workload.Name,
			Updated:   up,
			Status:    printStatus(w),
			Reason:    reason,
		})
	}
	return t.Flush()
}

func printStatus(w *models.ClustersDevMeshEnabledWorkload) string {
	var countOK, countMissing int64
	for _, c := range w.StatusCounts {
		switch c.Status {
		case "needs_update":
			return "NEEDS_UPDATE"
		case "missing":
			countMissing += 1
		case "ok":
			countOK += 1
		}
	}

	switch {
	case countOK > 0 && countMissing > 0:
		return "NEEDS_UPDATE"
	case countMissing > 0:
		return "MISSING"
	default:
		return "OK"
	}
}

func getUpdatedAndReason(w *models.ClustersDevMeshEnabledWorkload) (string, string) {
	var ok, missing, needsUpdate int64
	for _, c := range w.StatusCounts {
		switch c.Status {
		case "needs_update":
			needsUpdate += c.Count
		case "missing":
			missing += c.Count
		case "ok":
			ok += c.Count
		}
	}

	updated := fmt.Sprintf("%d/%d", ok, ok+missing+needsUpdate)
	var reasons []string
	if needsUpdate > 0 {
		reasons = append(reasons, fmt.Sprintf("%d pods without expected version", needsUpdate))
	}
	if missing > 0 {
		reasons = append(reasons, fmt.Sprintf("%d pods with missing sidecar", missing))
	}
	return updated, strings.Join(reasons, ", ")
}
