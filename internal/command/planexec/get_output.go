package planexec

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/client"
	planexecs "github.com/signadot/go-sdk/client/plan_executions"
	"github.com/spf13/cobra"
)

func newGetOutput(exec *config.PlanExecution) *cobra.Command {
	cfg := &config.PlanExecGetOutput{PlanExecution: exec}

	cmd := &cobra.Command{
		Use:   "get-output EXECUTION_ID [NAME]",
		Short: "Download a plan execution output",
		Long: `Download an output by name, or export all outputs to a directory.

Single output:
  signadot plan x get-output <exec-id> <name>          # plan-level output
  signadot plan x get-output <exec-id> <step>/<name>   # step-level output

Bulk export:
  signadot plan x get-output <exec-id> --all --dir ./outputs/
  signadot plan x get-output <exec-id> --all --dir ./outputs/ --metadata`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.All {
				if len(args) != 1 {
					return fmt.Errorf("--all expects exactly one argument (execution ID)")
				}
				if cfg.Dir == "" {
					return fmt.Errorf("--all requires --dir")
				}
				return getAllOutputs(cfg, cmd.ErrOrStderr(), args[0])
			}
			if len(args) != 2 {
				return fmt.Errorf("expected EXECUTION_ID and NAME arguments")
			}
			return getOutput(cfg, os.Stdout, args[0], args[1])
		},
	}

	cmd.Flags().BoolVar(&cfg.All, "all", false, "export all outputs")
	cmd.Flags().StringVar(&cfg.Dir, "dir", "", "directory to export outputs to (requires --all)")
	cmd.Flags().BoolVar(&cfg.Metadata, "metadata", false, "write metadata sidecar JSON files (requires --all)")

	return cmd
}

func getOutput(cfg *config.PlanExecGetOutput, out io.Writer, execID, name string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	transportCfg := cfg.GetBaseTransport()
	transportCfg.OverrideConsumers = true
	transportCfg.Consumers = map[string]runtime.Consumer{
		"*/*": runtime.ByteStreamConsumer(),
	}

	return cfg.APIClientWithCustomTransport(transportCfg,
		func(c *client.SignadotAPI) error {
			// If name contains '/', treat as step_id/output_name.
			if stepID, outputName, ok := strings.Cut(name, "/"); ok {
				params := planexecs.NewGetStepOutputParams().
					WithTimeout(4*time.Minute).
					WithOrgName(cfg.Org).
					WithExecutionID(execID).
					WithStepID(stepID).
					WithOutputName(outputName)
				_, _, err := c.PlanExecutions.GetStepOutput(params, nil, out)
				if err != nil {
					return fmt.Errorf("downloading output %q: %w", name, err)
				}
				return nil
			}

			params := planexecs.NewGetPlanExecutionOutputParams().
				WithTimeout(4*time.Minute).
				WithOrgName(cfg.Org).
				WithExecutionID(execID).
				WithOutputName(name)
			_, _, err := c.PlanExecutions.GetPlanExecutionOutput(params, nil, out)
			if err != nil {
				return fmt.Errorf("downloading output %q: %w", name, err)
			}
			return nil
		})
}

func getAllOutputs(cfg *config.PlanExecGetOutput, log io.Writer, execID string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	// Fetch execution to get output list.
	getParams := planexecs.NewGetPlanExecutionParams().
		WithOrgName(cfg.Org).
		WithExecutionID(execID)
	resp, err := cfg.Client.PlanExecutions.GetPlanExecution(getParams, nil)
	if err != nil {
		return err
	}

	outputs := resp.Payload.Status.Outputs
	if len(outputs) == 0 {
		fmt.Fprintln(log, "No outputs.")
		return nil
	}

	if err := os.MkdirAll(cfg.Dir, 0o755); err != nil {
		return err
	}

	transportCfg := cfg.GetBaseTransport()
	transportCfg.OverrideConsumers = true
	transportCfg.Consumers = map[string]runtime.Consumer{
		"*/*": runtime.ByteStreamConsumer(),
	}

	return cfg.APIClientWithCustomTransport(transportCfg,
		func(c *client.SignadotAPI) error {
			for _, o := range outputs {
				outPath := filepath.Join(cfg.Dir, o.Name)
				f, err := os.Create(outPath)
				if err != nil {
					return fmt.Errorf("creating %s: %w", outPath, err)
				}
				params := planexecs.NewGetPlanExecutionOutputParams().
					WithTimeout(4*time.Minute).
					WithOrgName(cfg.Org).
					WithExecutionID(execID).
					WithOutputName(o.Name)
				_, _, err = c.PlanExecutions.GetPlanExecutionOutput(params, nil, f)
				f.Close()
				if err != nil {
					return fmt.Errorf("downloading %q: %w", o.Name, err)
				}
				fmt.Fprintf(log, "Exported %s\n", outPath)

				// Write metadata sidecar if requested.
				if cfg.Metadata && o.Metadata != nil {
					metaPath := outPath + ".meta.json"
					metaJSON, err := json.MarshalIndent(o.Metadata, "", "  ")
					if err != nil {
						return fmt.Errorf("marshaling metadata for %q: %w", o.Name, err)
					}
					if err := os.WriteFile(metaPath, metaJSON, 0o644); err != nil {
						return fmt.Errorf("writing %s: %w", metaPath, err)
					}
					fmt.Fprintf(log, "Exported %s\n", metaPath)
				}
			}
			return nil
		})
}
