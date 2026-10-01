package config

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

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

// DryRunMode follows kubectl's --dry-run, so that the distinction between
// rendering locally and asking the server to validate is the one people already
// know.
type DryRunMode string

const (
	// DryRunNone applies normally.
	DryRunNone DryRunMode = "none"
	// DryRunClient renders and validates locally, without contacting the API.
	DryRunClient DryRunMode = "client"
	// DryRunServer additionally asks the API to validate. Not yet available.
	DryRunServer DryRunMode = "server"
)

func (m *DryRunMode) String() string {
	return string(*m)
}

// Set implements the pflag.Value interface.
func (m *DryRunMode) Set(v string) error {
	switch DryRunMode(v) {
	case DryRunNone, DryRunClient, DryRunServer:
		*m = DryRunMode(v)
		return nil
	default:
		return fmt.Errorf("unknown dry-run mode %q: expected one of none, client, server", v)
	}
}

// Type implements the pflag.Value interface.
func (m *DryRunMode) Type() string {
	return "none|client|server"
}

var (
	VarRx = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_.-]*$`)
)

type TemplateVal struct {
	Var, Val string
}

func (tv *TemplateVal) String() string {
	return tv.Var + "=" + tv.Val
}

type TemplateVals []TemplateVal

func (tvs *TemplateVals) Set(v string) error {
	varName, varVal, found := strings.Cut(v, "=")
	if !found {
		return fmt.Errorf("--set expects <var>=<val> syntax, got %q", v)
	}
	if !VarRx.MatchString(varName) {
		return fmt.Errorf("--set expects <var> to match %s, got %q", VarRx, varName)
	}
	*tvs = append(*tvs, TemplateVal{Var: varName, Val: varVal})
	return nil
}

func (tvs *TemplateVals) Type() string {
	return "string"
}

func (tvs *TemplateVals) String() string {
	b := bytes.NewBuffer(nil)
	for i := range *tvs {
		tv := &(*tvs)[i]
		if i != 0 {
			fmt.Fprintf(b, " ")
		}
		fmt.Fprintf(b, "--set %s", tv)
	}
	return b.String()
}
