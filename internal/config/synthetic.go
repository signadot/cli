package config

import "github.com/spf13/cobra"

type Synthetic struct {
	*API
}

type SyntheticApply struct {
	*Synthetic

	Filename     string
	TemplateVals TemplateVals
}

func (cfg *SyntheticApply) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&cfg.Filename, "filename", "f", "", "YAML or JSON file containing the synthetic test creation request")
	cmd.MarkFlagRequired("filename")
	cmd.Flags().Var(&cfg.TemplateVals, "set", "--set var=val")
}

type SyntheticGet struct {
	*Synthetic
}

type SyntheticList struct {
	*Synthetic

	// TODO query params
}

type SyntheticDelete struct {
	*Synthetic

	Filename     string
	TemplateVals TemplateVals
}

type SyntheticRun struct {
	*Synthetic

	Cluster    string
	Sandbox    string
	RouteGroup string
}

func (cfg *SyntheticRun) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&cfg.Cluster, "cluster", "c", "", "cluster name (required for test execution)")
	cmd.Flags().StringVarP(&cfg.Sandbox, "sandbox", "s", "", "sandbox")
	cmd.Flags().StringVarP(&cfg.RouteGroup, "routegroup", "r", "", "routegroup")
}
