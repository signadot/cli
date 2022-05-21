package print

import (
	"encoding/json"
	"io"

	"sigs.k8s.io/yaml"
)

func RawJSON(out io.Writer, v any) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func RawYAML(out io.Writer, v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	_, err = out.Write(data)
	return err
}
