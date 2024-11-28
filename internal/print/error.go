package print

import (
	"fmt"
	"github.com/signadot/cli/internal/config"
	"io"
)

func printErrorJson(err error) error {

	return nil
}

func Error(out io.Writer, err error, outputFormat config.OutputFormat) error {
	type errorResponse struct {
		Error string `json:"error"`
	}

	rawResponse := errorResponse{Error: err.Error()}

	switch outputFormat {
	case config.OutputFormatDefault:
		return err
	case config.OutputFormatJSON:
		return RawJSON(out, rawResponse)
	case config.OutputFormatYAML:
		return RawK8SYAML(out, rawResponse)
	default:
		return fmt.Errorf("unsupported output format: %q", outputFormat)
	}
}
