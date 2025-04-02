package config

import "github.com/spf13/cobra"

type HostedTest struct {
	*API
}

type HostedTestApply struct {
	*HostedTest

	Filename     string
	TemplateVals TemplateVals
}

func (cfg *HostedTestApply) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&cfg.Filename, "filename", "f", "", "YAML or JSON file containing the hosted test creation request")
	cmd.MarkFlagRequired("filename")
	cmd.Flags().Var(&cfg.TemplateVals, "set", "--set var=val")
}

type HostedTestGet struct {
	*HostedTest
}

type HostedTestList struct {
	*HostedTest

	// TODO query params
}

type HostedTestDelete struct {
	*HostedTest

	Filename     string
	TemplateVals TemplateVals
}

type HostedTestRun struct {
	*HostedTest

	Cluster    string
	Sandbox    string
	RouteGroup string
}

func (cfg *HostedTestRun) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&cfg.Cluster, "cluster", "c", "", "cluster name (required for test execution)")
	cmd.Flags().StringVarP(&cfg.Sandbox, "sandbox", "s", "", "sandbox")
	cmd.Flags().StringVarP(&cfg.RouteGroup, "routegroup", "r", "", "routegroup")
}
