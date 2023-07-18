package print

import (
	"encoding/json"
	"io"

	"github.com/goccy/go-yaml"
	k8syaml "sigs.k8s.io/yaml"
)

func RawJSON(out io.Writer, v any) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func RawYAML(out io.Writer, v any) error {
	opt := yaml.UseLiteralStyleIfMultiline(true)
	data, err := yaml.MarshalWithOptions(v, opt)
	if err != nil {
		return err
	}
	_, err = out.Write(data)
	return err
}

func RawK8SYAML(out io.Writer, v any) error {
	data, err := k8syaml.Marshal(v)
	if err != nil {
		return err
	}
	_, err = out.Write(data)
	return err
}
