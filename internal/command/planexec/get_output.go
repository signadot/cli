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
	"github.com/signadot/go-sdk/models"
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
  signadot plan x get-output <exec-id> --all --dir ./outputs/ --metadata=false  # skip sidecars`,
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
	cmd.Flags().BoolVar(&cfg.Metadata, "metadata", true,
		"write .meta.json sidecar files alongside each exported output (requires --all; pass --metadata=false to disable)")

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

	// Reuse collectAllOutputs to gather plan-level and step-level outputs.
	all := collectAllOutputs(resp.Payload)
	if len(all) == 0 {
		fmt.Fprintln(log, "No outputs.")
		return nil
	}

	// Build a metadata lookup keyed by qualified name (<name> for plan
	// outputs, <step>/<name> for step outputs). Outputs without any
	// explicit metadata map to nil entries — sidecars are still written
	// for them using the derived fields alone.
	metadataMap := buildMetadataMap(resp.Payload)

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
			for _, o := range all {
				// Determine file path: plan-level → <dir>/<name>, step-level → <dir>/<step>/<name>.
				var outPath string
				if o.Step != "" {
					stepDir := filepath.Join(cfg.Dir, o.Step)
					if err := os.MkdirAll(stepDir, 0o755); err != nil {
						return fmt.Errorf("creating %s: %w", stepDir, err)
					}
					outPath = filepath.Join(stepDir, o.Name)
				} else {
					outPath = filepath.Join(cfg.Dir, o.Name)
				}

				f, err := os.Create(outPath)
				if err != nil {
					return fmt.Errorf("creating %s: %w", outPath, err)
				}

				// Download using the appropriate API based on scope.
				// Capture Content-Type from the response so the sidecar
				// can record the server-supplied value rather than a
				// re-derived one (keeps the server's
				// plans.ResolveContentType as the single source of truth).
				qualName := o.Name
				var contentType string
				if o.Scope == "step" {
					qualName = o.Step + "/" + o.Name
					params := planexecs.NewGetStepOutputParams().
						WithTimeout(4*time.Minute).
						WithOrgName(cfg.Org).
						WithExecutionID(execID).
						WithStepID(o.Step).
						WithOutputName(o.Name)
					ok, partial, getErr := c.PlanExecutions.GetStepOutput(params, nil, f)
					err = getErr
					switch {
					case ok != nil:
						contentType = ok.ContentType
					case partial != nil:
						contentType = partial.ContentType
					}
				} else {
					params := planexecs.NewGetPlanExecutionOutputParams().
						WithTimeout(4*time.Minute).
						WithOrgName(cfg.Org).
						WithExecutionID(execID).
						WithOutputName(o.Name)
					ok, partial, getErr := c.PlanExecutions.GetPlanExecutionOutput(params, nil, f)
					err = getErr
					switch {
					case ok != nil:
						contentType = ok.ContentType
					case partial != nil:
						contentType = partial.ContentType
					}
				}
				f.Close()
				if err != nil {
					return fmt.Errorf("downloading %q: %w", qualName, err)
				}
				fmt.Fprintf(log, "Exported %s\n", outPath)

				// Write sidecar. --metadata is on by default; pass
				// --metadata=false to disable. The sidecar is always
				// emitted (regardless of whether the output carries
				// explicit metadata) so the derived fields — scope,
				// step, inline vs artifact, size, ready, contentType —
				// are visible to the reader without another round-trip.
				if cfg.Metadata {
					sidecar := outputSidecar{
						allOutput:   o,
						ContentType: contentType,
						Metadata:    metadataMap[qualName],
					}
					metaPath := outPath + ".meta.json"
					metaJSON, err := json.MarshalIndent(sidecar, "", "  ")
					if err != nil {
						return fmt.Errorf("marshaling sidecar for %q: %w", qualName, err)
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

// buildMetadataMap extracts metadata from plan-level and step-level outputs,
// keyed by "name" (plan-level) or "step/name" (step-level).
// buildMetadataMap returns each output's explicit metadata keyed by
// qualified name: "<name>" for plan-level outputs, "<step>/<name>" for
// step-level outputs. Outputs that carry no explicit metadata are
// absent from the map (so a lookup returns a nil map, which is the
// correct omitempty signal for the sidecar).
func buildMetadataMap(ex *models.PlanExecution) map[string]map[string]string {
	m := map[string]map[string]string{}
	if ex.Status == nil {
		return m
	}
	for _, o := range ex.Status.Outputs {
		if len(o.Metadata) > 0 {
			m[o.Name] = o.Metadata
		}
	}
	for _, s := range ex.Status.Steps {
		for _, o := range s.Outputs {
			if len(o.Metadata) > 0 {
				m[s.ID+"/"+o.Name] = o.Metadata
			}
		}
	}
	return m
}
