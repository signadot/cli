package plan

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/signadot/cli/internal/command/plantag"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	sdkplans "github.com/signadot/go-sdk/client/plans"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newCompile(plan *config.Plan) *cobra.Command {
	cfg := &config.PlanCompile{Plan: plan}

	cmd := &cobra.Command{
		Use:   "compile -f PROMPT_FILE",
		Short: "Compile a natural-language prompt into a runnable plan",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return compile(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}

	cfg.AddFlags(cmd)
	return cmd
}

func compile(cfg *config.PlanCompile, out, log io.Writer) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}

	// Read the prompt from file or stdin.
	var raw []byte
	var err error
	if cfg.Filename == "-" {
		raw, err = io.ReadAll(os.Stdin)
	} else {
		raw, err = os.ReadFile(cfg.Filename)
	}
	if err != nil {
		return fmt.Errorf("reading prompt: %w", err)
	}
	prompt := strings.TrimSpace(string(raw))
	if prompt == "" {
		return fmt.Errorf("prompt file %q is empty", cfg.Filename)
	}

	params := sdkplans.NewCompilePlanParams().
		WithOrgName(cfg.Org).
		WithData(&models.PlanCompileInput{
			Prompt: prompt,
		})
	resp, err := cfg.Client.Plans.CompilePlan(params, nil)
	if err != nil {
		return err
	}

	// If --tag was provided, tag the compiled plan.
	if cfg.Tag != "" {
		if _, err := plantag.ApplyTag(cfg.Plan, resp.Payload.ID, cfg.Tag); err != nil {
			return fmt.Errorf("plan compiled (id=%s) but tagging failed: %w", resp.Payload.ID, err)
		}
		if cfg.OutputFormat == config.OutputFormatDefault {
			fmt.Fprintf(log, "Tagged plan %s as %q\n", resp.Payload.ID, cfg.Tag)
		}
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printPlanDetails(out, resp.Payload)
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp.Payload)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp.Payload)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
