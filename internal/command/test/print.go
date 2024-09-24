package test

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/go-sdk/models"
)

func printTest(oFmt config.OutputFormat, w io.Writer, t *models.Test) error {
	switch oFmt {
	case config.OutputFormatDefault:
		// TODO
		return nil
	case config.OutputFormatJSON:
		return print.RawJSON(w, t)
	case config.OutputFormatYAML:
		return print.RawYAML(w, t)
	default:
		return fmt.Errorf("unsupported output format: %q", oFmt)
	}
}
