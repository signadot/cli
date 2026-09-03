package clio

import (
	"io"
	"os"

	"sigs.k8s.io/yaml"
)

// ReadFileOrStdin reads a file, treating the filename "-" as a placeholder
// meaning stdin.
func ReadFileOrStdin(filename string) ([]byte, error) {
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
	return io.ReadAll(in)
}

// LoadYAML unmarshals YAML (or JSON) from a file into the given type.
// It treats the filename "-" as a special placeholder meaning to use stdin.
func LoadYAML[T any](filename string) (*T, error) {
	data, err := ReadFileOrStdin(filename)
	if err != nil {
		return nil, err
	}

	var t T
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}
