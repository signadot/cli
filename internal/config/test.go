package config

import "github.com/spf13/cobra"

type Test struct {
	*API
}

type TestApply struct {
	*Test

	Filename     string
	TemplateVals TemplateVals
}

func (cfg *TestApply) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&cfg.Filename, "filename", "f", "", "YAML or JSON file containing the sandbox creation request")
	cmd.MarkFlagRequired("filename")
	cmd.Flags().Var(&cfg.TemplateVals, "set", "--set var=val")
}

type TestGet struct {
	*Test
}

type TestList struct {
	*Test

	// TODO query params
}

type TestDelete struct {
	*Test

	Filename     string
	TemplateVals TemplateVals
}

type RunTest struct {
	*Test
}
