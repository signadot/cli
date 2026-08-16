package plan

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/repoconfig"
	planexecs "github.com/signadot/go-sdk/client/plan_executions"
	plantags "github.com/signadot/go-sdk/client/plan_tags"
	"github.com/signadot/go-sdk/models"
)

// selectedPlan is one plan to run, with the tags that selected it. A plan can
// carry several tags, and more than one of them can match, so the tags are
// kept for reporting rather than as an identity.
type selectedPlan struct {
	plan *models.RunnablePlan
	tags []string
}

// label names the plan in output. The tags are what the author asked for, so
// they read better than an ID they never typed.
func (s *selectedPlan) label() string {
	if len(s.tags) == 0 {
		return s.plan.ID
	}
	return strings.Join(s.tags, ",")
}

// runsByTag reports whether this invocation runs a selected set rather than
// one named plan. Naming a plan — by ID or by --tag — always means exactly
// that plan, so the set path is only for an invocation that named neither.
func runsByTag(cfg *config.PlanRun, args []string) bool {
	return len(args) == 0 && cfg.Tag == ""
}

// runPlansByTag runs every plan the repository's configuration selects.
//
// Exit codes are the aggregate of the individual ones, with failure taking
// precedence over cancellation: 1 if any plan failed, else 2 if any was
// cancelled, else 0. A run that both failed and was cancelled is a failing run,
// because that is the outcome someone has to act on.
func runPlansByTag(ctx context.Context, cfg *config.PlanRun, out, log io.Writer, args []string) error {
	if cfg.Attach {
		return fmt.Errorf("--attach streams a single execution; it cannot be used when running plans by tag")
	}

	selector, err := repoconfig.NewPlanTagSelector(cfg.PlanTags, cfg.WithTags, cfg.WithoutTags)
	if err != nil {
		return err
	}

	selected, err := selectPlans(ctx, cfg, selector)
	if err != nil {
		return err
	}
	if len(selected) == 0 {
		return fmt.Errorf("no plans matched %s", strings.Join(selector.Patterns(), ", "))
	}

	if cfg.OutputFormat == config.OutputFormatDefault {
		fmt.Fprintf(log, "Running %s\n", plural(len(selected), "plan"))
	}

	execs := make([]*models.PlanExecution, 0, len(selected))
	for _, sel := range selected {
		exec, err := createExecutionFor(ctx, cfg, log, sel)
		if err != nil {
			// One plan that cannot start should not strand the rest: the
			// caller asked for a set, and a partial answer about the set is
			// worth more than none.
			fmt.Fprintf(log, "Could not start %s: %v\n", sel.label(), err)
			continue
		}
		execs = append(execs, exec)
	}
	if len(execs) == 0 {
		return fmt.Errorf("no executions could be started")
	}

	if !cfg.Wait {
		return writeTagRunOutput(cfg, out, log, selected, execs)
	}

	// Executions run concurrently on the runner group; this waits on each in
	// turn, which costs nothing but the order results are collected in.
	completed := make([]*models.PlanExecution, 0, len(execs))
	for i, exec := range execs {
		done, err := pollExecution(ctx, cfg, log, exec.ID)
		if err != nil {
			return err
		}
		completed = append(completed, done)

		if cfg.OutputDir != "" {
			if err := exportSelectedOutputs(cfg, log, selected[i], done); err != nil {
				fmt.Fprintf(log, "Warning: output export failed for %s: %v\n", selected[i].label(), err)
			}
		}
	}

	if err := writeTagRunOutput(cfg, out, log, selected, completed); err != nil {
		return err
	}
	exitForPhases(completed)
	return nil
}

// selectPlans resolves the selector against the org's tags, returning one
// entry per plan.
//
// Deduplication is by plan, not by tag. A plan can carry several tags — the
// tag table is unique on name, not on plan — so two patterns, or one pattern
// and one plan wearing two matching tags, would otherwise run it twice.
func selectPlans(ctx context.Context, cfg *config.PlanRun, selector *repoconfig.PlanTagSelector) ([]selectedPlan, error) {
	listParams := plantags.NewListPlanTagsParams().
		WithContext(ctx).
		WithOrgName(cfg.Org)
	resp, err := cfg.Client.PlanTags.ListPlanTags(listParams, nil)
	if err != nil {
		return nil, fmt.Errorf("listing plan tags: %w", err)
	}

	byName := make(map[string]*models.PlanTag, len(resp.Payload))
	names := make([]string, 0, len(resp.Payload))
	for _, tag := range resp.Payload {
		if tag == nil || tag.Name == "" {
			continue
		}
		byName[tag.Name] = tag
		names = append(names, tag.Name)
	}

	matched, err := selector.Select(names)
	if err != nil {
		return nil, err
	}
	return groupSelectedByPlan(matched, byName), nil
}

// groupSelectedByPlan collapses matched tag names to the plans they name.
//
// This is the dedupe that matters. The tag table is unique on tag name, not on
// plan, so one plan can wear several tags and more than one of them can match —
// from two configured patterns, or from one pattern matching two of its tags.
// Without collapsing, that plan would be run once per matching tag.
//
// A tag naming no plan is skipped rather than reported: its plan was deleted,
// and the rest of the set is still worth running.
func groupSelectedByPlan(matched []string, byName map[string]*models.PlanTag) []selectedPlan {
	byPlan := make(map[string]*selectedPlan, len(matched))
	order := make([]string, 0, len(matched))
	for _, name := range matched {
		tag, ok := byName[name]
		if !ok || tag.Plan == nil || tag.Plan.ID == "" {
			continue
		}
		sel, ok := byPlan[tag.Plan.ID]
		if !ok {
			byPlan[tag.Plan.ID] = &selectedPlan{plan: tag.Plan, tags: []string{name}}
			order = append(order, tag.Plan.ID)
			continue
		}
		sel.tags = append(sel.tags, name)
	}

	sort.Strings(order)
	selected := make([]selectedPlan, 0, len(order))
	for _, planID := range order {
		sel := byPlan[planID]
		sort.Strings(sel.tags)
		selected = append(selected, *sel)
	}
	return selected
}

func createExecutionFor(ctx context.Context, cfg *config.PlanRun, log io.Writer,
	sel selectedPlan) (*models.PlanExecution, error) {
	params := buildParams(cfg.Params)
	if params == nil && (cfg.Sandbox != "" || cfg.RouteGroup != "") {
		params = make(map[string]any)
	}
	if err := applyRoutingFlags(cfg, sel.plan.Spec, params); err != nil {
		return nil, err
	}

	secrets := buildSecrets(cfg.Secrets)
	for name := range secrets {
		if _, ok := params[name]; ok {
			return nil, fmt.Errorf("param %q appears in both --param and --param-secret; specify only one", name)
		}
	}

	createParams := planexecs.NewCreatePlanExecutionParams().
		WithContext(ctx).
		WithOrgName(cfg.Org).
		WithData(&models.PlanExecutionSpec{
			PlanID:  sel.plan.ID,
			Cluster: cfg.Cluster,
			Params:  params,
			Secrets: secrets,
		})
	resp, err := cfg.Client.PlanExecutions.CreatePlanExecution(createParams, nil)
	if err != nil {
		return nil, err
	}
	if cfg.OutputFormat == config.OutputFormatDefault {
		fmt.Fprintf(log, "Created execution %s for %s\n", resp.Payload.ID, sel.label())
	}
	return resp.Payload, nil
}

// exportSelectedOutputs gives each plan its own directory, since several plans
// writing an output of the same name into one directory would overwrite each
// other.
func exportSelectedOutputs(cfg *config.PlanRun, log io.Writer, sel selectedPlan,
	exec *models.PlanExecution) error {
	perPlan := *cfg
	perPlan.OutputDir = filepath.Join(cfg.OutputDir, sanitizeDirName(sel.label()))
	if err := os.MkdirAll(perPlan.OutputDir, 0755); err != nil {
		return err
	}
	return exportOutputs(&perPlan, log, exec)
}

// sanitizeDirName keeps a tag usable as a directory name. Tag names allow ':'
// and '+', which are legal on the filesystems we target but awkward enough in
// a path to be worth flattening.
func sanitizeDirName(name string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '/', ':', '+', ' ':
			return '-'
		}
		return r
	}, name)
}

func writeTagRunOutput(cfg *config.PlanRun, out, log io.Writer,
	selected []selectedPlan, execs []*models.PlanExecution) error {
	switch cfg.OutputFormat {
	case config.OutputFormatJSON:
		return print.RawJSON(out, execs)
	case config.OutputFormatYAML:
		return print.RawYAML(out, execs)
	}

	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "TAGS\tPLAN\tEXECUTION\tPHASE")
	for i, exec := range execs {
		phase := ""
		if exec.Status != nil {
			phase = string(exec.Status.Phase)
		}
		label := exec.Spec.PlanID
		if i < len(selected) {
			label = selected[i].label()
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", label, exec.Spec.PlanID, exec.ID, phase)
	}
	return tw.Flush()
}

// exitForPhases exits with the aggregate result. It is the multi-plan
// counterpart of the switch at the end of runPlan, and keeps the same meanings
// for 1 and 2.
func exitForPhases(execs []*models.PlanExecution) {
	var failed, cancelled bool
	for _, exec := range execs {
		if exec.Status == nil {
			continue
		}
		switch exec.Status.Phase {
		case models.PlansExecutionPhaseFailed:
			failed = true
		case models.PlansExecutionPhaseCancelled:
			cancelled = true
		}
	}
	switch {
	case failed:
		os.Exit(1)
	case cancelled:
		os.Exit(2)
	}
}

func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
