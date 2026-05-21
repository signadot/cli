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

	// Flags
	AllVersions bool
}

func (c *ResourcePluginList) AddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&c.AllVersions, "all-versions", "A", false,
		"return every published version of every plugin (one row per name+version), sorted by name then semver-descending; without this flag, list returns only the highest-semver version of each plugin")
}

type ResourcePluginVersions struct {
	*ResourcePlugin
}
