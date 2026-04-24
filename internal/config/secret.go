package config

import "github.com/spf13/cobra"

type Secret struct {
	*API
}

type SecretCreate struct {
	*Secret

	// Flags
	Value        string
	ValueFile    string
	ValueStdin   bool
	Description  string
	Filename     string
	TemplateVals TemplateVals
}

func (c *SecretCreate) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.Value, "value", "", "secret value as a literal string (leaks into shell history)")
	cmd.Flags().StringVar(&c.ValueFile, "value-file", "", "path to a file whose contents become the secret value")
	cmd.Flags().BoolVar(&c.ValueStdin, "value-stdin", false, "read the secret value from stdin")
	cmd.Flags().StringVar(&c.Description, "description", "", "human-readable description")
	cmd.Flags().StringVarP(&c.Filename, "filename", "f", "", "YAML or JSON file containing the secret (fields: name, value, description)")
	cmd.Flags().Var(&c.TemplateVals, "set", "--set var=val (used with -f)")
}

type SecretUpdate struct {
	*Secret

	// Flags
	Value        string
	ValueFile    string
	ValueStdin   bool
	Description  string
	Filename     string
	TemplateVals TemplateVals
}

func (c *SecretUpdate) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.Value, "value", "", "new secret value as a literal string (leaks into shell history)")
	cmd.Flags().StringVar(&c.ValueFile, "value-file", "", "path to a file whose contents become the new secret value")
	cmd.Flags().BoolVar(&c.ValueStdin, "value-stdin", false, "read the new secret value from stdin")
	cmd.Flags().StringVar(&c.Description, "description", "", "new human-readable description")
	cmd.Flags().StringVarP(&c.Filename, "filename", "f", "", "YAML or JSON file containing the secret (fields: name, value, description)")
	cmd.Flags().Var(&c.TemplateVals, "set", "--set var=val (used with -f)")
}

type SecretGet struct {
	*Secret
}

type SecretList struct {
	*Secret
}

type SecretDelete struct {
	*Secret

	// Flags
	Filename     string
	TemplateVals TemplateVals
}

func (c *SecretDelete) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Filename, "filename", "f", "", "optional YAML or JSON file containing the original secret (name is read from it)")
	cmd.Flags().Var(&c.TemplateVals, "set", "--set var=val")
}
