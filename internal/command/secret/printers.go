package secret

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
)

type secretRow struct {
	Name        string `sdtab:"NAME"`
	Description string `sdtab:"DESCRIPTION"`
	Created     string `sdtab:"CREATED"`
	Updated     string `sdtab:"UPDATED"`
}

func printSecretTable(out io.Writer, secrets []*models.Secret) error {
	t := sdtab.New[secretRow](out)
	t.AddHeader()
	for _, s := range secrets {
		t.AddRow(secretRow{
			Name:        s.Name,
			Description: s.Description,
			Created:     s.CreatedAt,
			Updated:     s.UpdatedAt,
		})
	}
	return t.Flush()
}

func printSecretDetails(out io.Writer, s *models.Secret) error {
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	fmt.Fprintf(tw, "Name:\t%s\n", s.Name)
	if s.Description != "" {
		fmt.Fprintf(tw, "Description:\t%s\n", s.Description)
	}
	fmt.Fprintf(tw, "Created:\t%s\n", utils.FormatTimestamp(s.CreatedAt))
	if s.UpdatedAt != "" {
		fmt.Fprintf(tw, "Updated:\t%s\n", utils.FormatTimestamp(s.UpdatedAt))
	}
	return tw.Flush()
}
