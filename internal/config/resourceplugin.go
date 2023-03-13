package config

import (
	"github.com/spf13/cobra"
)

type ResourcePlugin struct {
	*API
}

type ResourcePluginApply struct {
	*ResourcePlugin

	// Flags
	Filename     string
	TemplateVals TemplateVals
}

func (c *ResourcePluginApply) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Filename, "filename", "f", "", "YAML or JSON file containing the resource plugin creation request")
	cmd.MarkFlagRequired("filename")
	cmd.Flags().Var(&c.TemplateVals, "set", "--set var=val")
}

type ResourcePluginDelete struct {
	*ResourcePlugin

	// Flags
	Filename     string
	TemplateVals TemplateVals
}

func (c *ResourcePluginDelete) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Filename, "filename", "f", "", "optional YAML or JSON file containing the original resource plugin creation request")
	cmd.Flags().Var(&c.TemplateVals, "set", "--set var=val")
}

type ResourcePluginGet struct {
	*ResourcePlugin
}

type ResourcePluginList struct {
	*ResourcePlugin
}
