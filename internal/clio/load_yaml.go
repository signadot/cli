package clio

import (
	"io"
	"os"

	"sigs.k8s.io/yaml"
)

// LoadYAML unmarshals YAML (or JSON) from a file into the given type.
// It treats the filename "-" as a special placeholder meaning to use stdin.
func LoadYAML[T any](filename string) (*T, error) {
	var in io.Reader
	if filename == "-" {
		in = os.Stdin
	} else {
		file, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		in = file
	}

	data, err := io.ReadAll(in)
	if err != nil {
		return nil, err
	}

	var t T
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}
