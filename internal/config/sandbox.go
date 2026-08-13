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
	Filename      string
	Wait          bool
	WaitTimeout   time.Duration
	TemplateVals  TemplateVals
	DryRun        DryRunMode
	Patch         string
	Name          string
	TTL           string
	CIContext     string
	DefaultLabels bool
}

func (c *SandboxApply) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Filename, "filename", "f", "", "YAML or JSON file containing the sandbox creation request, or a values document")
	cmd.Flags().BoolVar(&c.Wait, "wait", true, "wait for the sandbox status to be Ready before returning")
	cmd.Flags().DurationVar(&c.WaitTimeout, "wait-timeout", 3*time.Minute, "timeout when waiting for the sandbox to be Ready")
	cmd.MarkFlagRequired("filename")
	cmd.Flags().Var(&c.TemplateVals, "set", "--set var=val")

	c.DryRun = DryRunNone
	cmd.Flags().Var(&c.DryRun, "dry-run", "print the rendered sandbox spec instead of applying it: \"client\" renders and validates locally")
	cmd.Flags().StringVar(&c.Patch, "patch", "", "YAML or JSON file containing a merge patch to apply to the rendered spec")
	cmd.Flags().StringVar(&c.Name, "name", "", "sandbox name, overriding any name in the file")
	cmd.Flags().StringVar(&c.TTL, "ttl", "", "sandbox time to live, e.g. 4h, overriding any TTL in the file")
	cmd.Flags().StringVar(&c.CIContext, "ci-context", "auto", "CI system to take the sandbox name and built-in labels from: auto, github or none")
	cmd.Flags().BoolVar(&c.DefaultLabels, "default-labels", true, "add the built-in signadot/* labels when a CI context is detected")
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

type SandboxGetFiles struct {
	*Sandbox
	Local     string
	Container string
	OutputDir string
	NoClobber bool
}

func (c *SandboxGetFiles) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Local, "local", "l", "", "local workload name, defaults to the first local workload in the sandbox")
	cmd.Flags().StringVarP(&c.Container, "container", "c", "", "container name, defaults to the first container in the local workload")
	cmd.Flags().BoolVar(&c.NoClobber, "no-clobber", false, "do not overwrite files")
	cmd.Flags().StringVarP(&c.OutputDir, "output-dir", "d", "", "output directory")
}

type SandboxCleanFiles struct {
	*Sandbox
}

type SandboxGetEnv struct {
	*Sandbox
	Local      string
	Container  string
	ShowSource bool
}

func (c *SandboxGetEnv) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&c.Local, "local", "l", "", "local workload name, defaults to the first local workload in the sandbox")
	cmd.Flags().StringVarP(&c.Container, "container", "c", "", "container name, defaults to the first container in the local workload")
	cmd.Flags().BoolVarP(&c.ShowSource, "show-source", "s", false, "show source in comments")
}
