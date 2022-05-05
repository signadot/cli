package config

import "fmt"

type OutputFormat string

const (
	OutputFormatDefault OutputFormat = ""
	OutputFormatYAML    OutputFormat = "yaml"
	OutputFormatJSON    OutputFormat = "json"
)

func (o *OutputFormat) String() string {
	return string(*o)
}

// Set implements the pflag.Value interface.
func (o *OutputFormat) Set(v string) error {
	switch OutputFormat(v) {
	case OutputFormatDefault, OutputFormatYAML, OutputFormatJSON:
		*o = OutputFormat(v)
	default:
		return fmt.Errorf("unknown output format: %v", v)
	}
	return nil
}

// Type implements the pflag.Value interface.
func (o *OutputFormat) Type() string {
	return "string"
}
