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

type SandboxGetFiles struct {
  *Sandbox
  Local     string
  Container string
  OutputDir string
  NoClobber bool
}

func (c *SandboxGetFiles) AddFlags(cmd *cobra.Command) {
  cmd.Flags().StringVarP(&c.Local, "local", "l", "", "local workload name (default to first)")
  cmd.Flags().StringVarP(&c.Container, "container", "c", "", "container name (defaults to first)")
  cmd.Flags().BoolVar(&c.NoClobber, "no-clobber", false, "do not overwrite files")
  cmd.Flags().StringVarP(&c.OutputDir, "output-dir", "d", "", "output directory")
}

type SandboxCleanFiles struct {
  *Sandbox
}

type SandboxGetEnv struct {
  *Sandbox
  Local     string
  Container string
}

func (c *SandboxGetEnv) AddFlags(cmd *cobra.Command) {
  cmd.Flags().StringVarP(&c.Local, "local", "l", "", "local workload name (default to first)")
  cmd.Flags().StringVarP(&c.Container, "container", "c", "", "container")
}

type SandboxCreate struct {
  *Sandbox

  // Flags
  Cluster            string
  KubernetesWorkload string
  TTL                string
  Wait               bool
  WaitTimeout        time.Duration
}

func (c *SandboxCreate) AddFlags(cmd *cobra.Command) {
  cmd.Flags().StringVar(&c.Cluster, "cluster", "", "specify cluster connection config")
  cmd.Flags().StringVar(&c.KubernetesWorkload, "kubernetes-workload", "", "Kubernetes workload in format kind/namespace/name (e.g., deployment/default/myapp)")
  cmd.Flags().StringVar(&c.TTL, "ttl", "", "Time to live for the sandbox (e.g., 20h, 30m, 1d)")
  cmd.Flags().BoolVar(&c.Wait, "wait", true, "wait for the sandbox status to be Ready before returning")
  cmd.Flags().DurationVar(&c.WaitTimeout, "wait-timeout", 3*time.Minute, "timeout when waiting for the sandbox to be Ready")
}

type SandboxSetImage struct {
  *Sandbox

  // Flags
  Workload string
  Image    string
}

func (c *SandboxSetImage) AddFlags(cmd *cobra.Command) {
  cmd.Flags().StringVar(&c.Workload, "workload", "", "workload name to set image for")
  cmd.MarkFlagRequired("workload")
}

type SandboxSetEnv struct {
  *Sandbox

  // Flags
  Workload string
  EnvVars  []string
}

func (c *SandboxSetEnv) AddFlags(cmd *cobra.Command) {
  cmd.Flags().StringVar(&c.Workload, "workload", "", "workload name to set environment variables for")
  cmd.MarkFlagRequired("workload")
}
