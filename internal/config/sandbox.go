package config

import (
	"time"

	"github.com/spf13/cobra"
)

type Sandbox struct {
	*API
}

type SandboxApply struct {
	*Sandbox

	// Flags
	Filename     string
	Wait         bool
	WaitTimeout  time.Duration
	TemplateVals TemplateVals
}

func (c *SandboxApply) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Filename, "filename", "f", "", "YAML or JSON file containing the sandbox creation request")
	cmd.Flags().BoolVar(&c.Wait, "wait", true, "wait for the sandbox status to be Ready before returning")
	cmd.Flags().DurationVar(&c.WaitTimeout, "wait-timeout", 3*time.Minute, "timeout when waiting for the sandbox to be Ready")
	cmd.MarkFlagRequired("filename")
	cmd.Flags().Var(&c.TemplateVals, "set", "--set var=val")
}

type SandboxDelete struct {
	*Sandbox

	// Flags
	Filename     string
	Wait         bool
	WaitTimeout  time.Duration
	TemplateVals TemplateVals
	Force        bool
}

func (c *SandboxDelete) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Filename, "filename", "f", "", "optional YAML or JSON file containing the original sandbox creation request")
	cmd.Flags().BoolVar(&c.Wait, "wait", true, "wait for the sandbox to finish terminating before returning")
	cmd.Flags().DurationVar(&c.WaitTimeout, "wait-timeout", 5*time.Minute, "timeout when waiting for the sandbox to finish terminating")
	cmd.Flags().BoolVar(&c.Force, "force", false, "force delete the sandbox, removing resources without deprovisioning them")
	cmd.Flags().Var(&c.TemplateVals, "set", "--set var=val")
}

type SandboxGet struct {
	*Sandbox
}

type SandboxList struct {
	*Sandbox
}

type SandboxMakeLocal struct {
	*Sandbox

	// Flags
	Cluster      string
	Workload     string
	Unprivileged bool
	PortMappings []string

	Wait        bool
	WaitTimeout time.Duration
}

func (c *SandboxMakeLocal) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Workload, "workload", "w", "", "workload to be forked")
	cmd.Flags().StringVar(&c.Cluster, "cluster", "", "cluster associated with the sandbox")
	cmd.Flags().StringArrayVar(&c.PortMappings, "port-mapping", []string{}, "workload to be forked")

	cmd.Flags().BoolVar(&c.Unprivileged, "unprivileged", false, "Provide mappings from workload ports to tcp addresses to which the local workstation can listen")

	cmd.Flags().BoolVar(&c.Wait, "wait", true, "wait for the sandbox status to be Ready before returning")
	cmd.Flags().DurationVar(&c.WaitTimeout, "wait-timeout", 3*time.Minute, "timeout when waiting for the sandbox to be Ready")

	cmd.MarkFlagRequired("cluster")
}
