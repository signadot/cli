package print

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/models"
)

type endpointRow struct {
	Desc string `sdtab:"PREVIEW ENDPOINT"`
	URL  string `sdtab:"URL"`
}

func PreviewEndpointTable(out io.Writer, endpoints []*models.PreviewEndpoint) error {
	t := sdtab.New[endpointRow](out)
	t.AddHeader()
	for _, ep := range endpoints {
		desc := ep.Name
		if ep.ForkOf != nil {
			desc = fmt.Sprintf("Fork of %s/%s", *ep.ForkOf.Namespace, *ep.ForkOf.Name)
		}
		t.AddRow(endpointRow{
			Desc: desc,
			URL:  ep.PreviewURL,
		})
	}
	return t.Flush()
}
