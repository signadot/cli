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

type SandboxRender struct {
	*Sandbox

	// Input selection
	Template     string
	Filename     string
	ValuesFile   string
	TemplateVals TemplateVals
	PatchFile    string

	// Fork sugar / shared overrides
	Cluster       string
	Namespace     string
	Forks         []string
	Kind          string
	Image         string
	ImageTemplate string
	Name          string
	TTL           string

	// CI context detection and validation
	Context  string
	Validate string
}

func (c *SandboxRender) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.Template, "template", "", "built-in template to render (e.g. fork-deployment or fork-deployment@v1)")
	cmd.Flags().StringVarP(&c.Filename, "filename", "f", "", "user template file using @{var} placeholders (mutually exclusive with --template)")
	cmd.Flags().StringVar(&c.ValuesFile, "values", "", "values document (schema v1) file, or '-' for stdin")
	cmd.Flags().Var(&c.TemplateVals, "set", "--set var=val (for -f template files)")
	cmd.Flags().StringVar(&c.PatchFile, "patch", "", "YAML merge patch applied last to the rendered spec")

	cmd.Flags().StringVar(&c.Cluster, "cluster", "", "target cluster name")
	cmd.Flags().StringVar(&c.Namespace, "namespace", "", "default namespace for forks")
	cmd.Flags().StringArrayVar(&c.Forks, "fork", nil, "workload to fork; repeatable. Optional inline attrs: name,image=...,namespace=...,kind=...")
	cmd.Flags().StringVar(&c.Kind, "kind", "Deployment", "default kind for forks that omit one")
	cmd.Flags().StringVar(&c.Image, "image", "", "image for the forked workload (single-fork only)")
	cmd.Flags().StringVar(&c.ImageTemplate, "image-template", "", "image applied to forks without an explicit image, e.g. ghcr.io/acme/{workload}:{sha}")
	cmd.Flags().StringVar(&c.Name, "name", "", "override the (default deterministic) sandbox name")
	cmd.Flags().StringVar(&c.TTL, "ttl", "", "sandbox TTL (e.g. 2d); defaults applied in CI context")

	cmd.Flags().StringVar(&c.Context, "context", "auto", "CI context detection: auto|none|github|gitlab|circleci")
	cmd.Flags().StringVar(&c.Validate, "validate", "client", "validation mode: client|none")
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
