package repoconfig

import (
	"reflect"
	"testing"
)

func selector(t *testing.T, patterns, with, without []string) *PlanTagSelector {
	t.Helper()
	// Patterns given explicitly, so no repository configuration is consulted.
	sel, err := NewPlanTagSelector(patterns, with, without)
	if err != nil {
		t.Fatalf("NewPlanTagSelector failed: %v", err)
	}
	return sel
}

func TestPlanTagSelector_Select(t *testing.T) {
	tags := []string{
		"checkout-smoke",
		"checkout-full",
		"billing-smoke",
		"suite:integration",
		"suite:unit",
	}

	cases := []struct {
		name     string
		patterns []string
		with     []string
		without  []string
		want     []string
	}{
		{
			name:     "exact name",
			patterns: []string{"checkout-smoke"},
			want:     []string{"checkout-smoke"},
		},
		{
			name:     "glob widens",
			patterns: []string{"checkout-*"},
			want:     []string{"checkout-full", "checkout-smoke"},
		},
		{
			name:     "several patterns union",
			patterns: []string{"checkout-smoke", "billing-*"},
			want:     []string{"billing-smoke", "checkout-smoke"},
		},
		{
			// ':' is a legal tag character, so a key:value convention works
			// without the server knowing about it.
			name:     "colon convention",
			patterns: []string{"suite:*"},
			want:     []string{"suite:integration", "suite:unit"},
		},
		{
			name:     "without removes",
			patterns: []string{"checkout-*"},
			without:  []string{"*-full"},
			want:     []string{"checkout-smoke"},
		},
		{
			// Every --with-tag must match, unlike patterns which union.
			name:     "with narrows and all must match",
			patterns: []string{"*"},
			with:     []string{"checkout-*", "*-smoke"},
			want:     []string{"checkout-smoke"},
		},
		{
			name:     "nothing matches",
			patterns: []string{"nope-*"},
			want:     nil,
		},
		{
			name:     "without beats pattern",
			patterns: []string{"checkout-smoke"},
			without:  []string{"checkout-smoke"},
			want:     nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := selector(t, c.patterns, c.with, c.without).Select(tags)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
}

// The result is sorted and free of repeats however the input arrived, so the
// same configuration selects the same set in the same order every run.
func TestPlanTagSelector_SelectIsStable(t *testing.T) {
	sel := selector(t, []string{"*"}, nil, nil)
	got, err := sel.Select([]string{"b", "a", "b", "c", "a"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// A bad glob is the author's mistake and should say so, rather than quietly
// selecting nothing.
func TestPlanTagSelector_InvalidPattern(t *testing.T) {
	for _, c := range []struct {
		name                    string
		patterns, with, without []string
	}{
		{name: "pattern", patterns: []string{"[bad"}},
		{name: "with", patterns: []string{"*"}, with: []string{"[bad"}},
		{name: "without", patterns: []string{"*"}, without: []string{"[bad"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := selector(t, c.patterns, c.with, c.without).Select([]string{"anything"})
			if err == nil {
				t.Fatal("expected an error for a malformed glob")
			}
		})
	}
}

// Patterns are reported so a run that matched nothing can say what it was
// looking for.
func TestPlanTagSelector_Patterns(t *testing.T) {
	sel := selector(t, []string{"checkout-*", "billing-*"}, nil, nil)
	if got := sel.Patterns(); !reflect.DeepEqual(got, []string{"checkout-*", "billing-*"}) {
		t.Fatalf("got %v", got)
	}
}
