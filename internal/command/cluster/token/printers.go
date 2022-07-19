package token

import (
	"io"

	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/models"
)

type tokenRow struct {
	ID              string `sdtab:"ID"`
	Token           string `sdtab:"TOKEN"`
	Created         string `sdtab:"CREATED"`
	LastConnectedAt string `sdtab:"LAST CONNECTED"`
	LastConnectedIP string `sdtab:"LAST CONNECTED IP"`
}

func printTokenTable(out io.Writer, tokens []*models.ClusterToken) error {
	t := sdtab.New[tokenRow](out)
	t.AddHeader()
	for _, token := range tokens {
		t.AddRow(tokenRow{
			ID:              token.ID,
			Token:           token.MaskedValue + "...",
			Created:         token.CreatedAt,
			LastConnectedAt: token.Status.LastConnectedAt,
			LastConnectedIP: token.Status.LastConnectedIP,
		})
	}
	return t.Flush()
}
