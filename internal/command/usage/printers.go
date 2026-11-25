package usage

import (
	"io"
	"strconv"

	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/models"
)

type usageRow struct {
	Name    string `sdtab:"NAME"`
	Kind    string `sdtab:"KIND"`
	Used    string `sdtab:"USED"`
	Cap     string `sdtab:"CAP"`
	Percent string `sdtab:"%USED"`
	Resets  string `sdtab:"RESETS"`
}

func printUsageTable(out io.Writer, resp *models.UsageResponse) error {
	t := sdtab.New[usageRow](out)
	t.AddHeader()
	for _, it := range resp.Items {
		t.AddRow(usageRow{
			Name:    name(it),
			Kind:    string(it.Kind),
			Used:    strconv.FormatInt(it.Used, 10),
			Cap:     capCell(it),
			Percent: percentCell(it),
			Resets:  resetsCell(it),
		})
	}
	return t.Flush()
}

// name prefers the human display name, falling back to the stable id so a row
// is never blank.
func name(it *models.UsageItem) string {
	if it.DisplayName != "" {
		return it.DisplayName
	}
	return it.ID
}

// capCell shows the cap, or "unlimited" when the plan sets no cap. A zero cap
// means the plan does not include this item.
func capCell(it *models.UsageItem) string {
	if it.Unlimited {
		return "unlimited"
	}
	if it.Cap == nil {
		return "-"
	}
	return strconv.FormatInt(*it.Cap, 10)
}

// percentCell is blank when there is no cap to measure against (unlimited, or a
// cap of 0), so the column never shows a misleading +Inf% or 0%.
func percentCell(it *models.UsageItem) string {
	if it.PercentUsed == nil {
		return ""
	}
	return strconv.FormatFloat(*it.PercentUsed, 'f', -1, 64) + "%"
}

// resetsCell shows when a quota's window resets; limits have no window.
func resetsCell(it *models.UsageItem) string {
	if it.Window == nil {
		return ""
	}
	return it.Window.End
}
