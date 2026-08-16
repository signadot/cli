package plan

import (
	"reflect"
	"testing"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/models"
)

func TestRunsByTag(t *testing.T) {
	cases := []struct {
		name string
		cfg  *config.PlanRun
		args []string
		want bool
	}{
		{"no plan named", &config.PlanRun{}, nil, true},
		{"plan ID names one plan", &config.PlanRun{}, []string{"pl-1"}, false},
		{"--tag names one plan", &config.PlanRun{Tag: "checkout"}, nil, false},
		{"--tags selects a set", &config.PlanRun{PlanTags: []string{"checkout-*"}}, nil, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := runsByTag(c.cfg, c.args); got != c.want {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestSelectedPlanLabel(t *testing.T) {
	withTags := &selectedPlan{plan: &models.RunnablePlan{ID: "pl-1"}, tags: []string{"a", "b"}}
	if got := withTags.label(); got != "a,b" {
		t.Errorf("got %q, want the tags", got)
	}
	// A plan reached without a tag has only its ID to go by.
	noTags := &selectedPlan{plan: &models.RunnablePlan{ID: "pl-1"}}
	if got := noTags.label(); got != "pl-1" {
		t.Errorf("got %q, want the plan ID", got)
	}
}

// Tag names are legal in ways directory names are awkward with.
func TestSanitizeDirName(t *testing.T) {
	cases := map[string]string{
		"checkout-smoke":    "checkout-smoke",
		"suite:integration": "suite-integration",
		"a+b":               "a-b",
		"a:b+c":             "a-b-c",
	}
	for in, want := range cases {
		if got := sanitizeDirName(in); got != want {
			t.Errorf("sanitizeDirName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPlural(t *testing.T) {
	if got := plural(1, "plan"); got != "1 plan" {
		t.Errorf("got %q", got)
	}
	if got := plural(2, "plan"); got != "2 plans" {
		t.Errorf("got %q", got)
	}
}

// groupByPlan is the dedupe that matters: the tag table is unique on tag name,
// not on plan, so a plan can wear several tags and several of them can match.
// Running by tag without collapsing to plans would run one plan twice.
func TestGroupSelectedByPlan(t *testing.T) {
	planA := &models.RunnablePlan{ID: "pl-a"}
	planB := &models.RunnablePlan{ID: "pl-b"}

	byName := map[string]*models.PlanTag{
		"checkout-smoke":    {Name: "checkout-smoke", Plan: planA},
		"suite:integration": {Name: "suite:integration", Plan: planA},
		"billing-smoke":     {Name: "billing-smoke", Plan: planB},
		"orphan":            {Name: "orphan"},
	}

	got := groupSelectedByPlan([]string{"billing-smoke", "checkout-smoke", "orphan", "suite:integration"}, byName)

	want := []selectedPlan{
		{plan: planA, tags: []string{"checkout-smoke", "suite:integration"}},
		{plan: planB, tags: []string{"billing-smoke"}},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d plans, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].plan.ID != want[i].plan.ID {
			t.Errorf("plan %d: got %q, want %q", i, got[i].plan.ID, want[i].plan.ID)
		}
		if !reflect.DeepEqual(got[i].tags, want[i].tags) {
			t.Errorf("plan %d tags: got %v, want %v", i, got[i].tags, want[i].tags)
		}
	}
}

// A tag whose plan was deleted selects nothing rather than erroring: the rest
// of the set is still worth running.
func TestGroupSelectedByPlan_OrphanTagOnly(t *testing.T) {
	byName := map[string]*models.PlanTag{"orphan": {Name: "orphan"}}
	if got := groupSelectedByPlan([]string{"orphan"}, byName); len(got) != 0 {
		t.Fatalf("got %+v, want nothing", got)
	}
}
