package sandbox

import (
	"github.com/signadot/cli/internal/sdtab"
	"github.com/signadot/go-sdk/models"
)

type sbRow struct {
	models.SandboxInfo
	status string
}

func (r *sbRow) Columns() []sdtab.Column[*sbRow] {
	return []sdtab.Column[*sbRow]{
		{
			Title: "NAME",
			Get:   func(r *sbRow) string { return r.Name },
		},
		{
			Title:    "DESCRIPTION",
			Get:      func(r *sbRow) string { return r.Description },
			Truncate: true,
		},
		{
			Title: "CLUSTER",
			Get:   func(r *sbRow) string { return r.ClusterName },
		},
		{
			Title:    "CREATED",
			Get:      func(r *sbRow) string { return r.CreatedAt },
			Truncate: true,
		},
		{
			Title: "STATUS",
			Get:   func(r *sbRow) string { return r.status },
		},
	}
}
