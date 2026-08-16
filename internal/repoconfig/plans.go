package repoconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// PlanTagSelector decides which plan tags a repository's configuration
// selects, and is the plan counterpart of TestFinder.
//
// The shapes differ because the things being selected do. A smart test is a
// file in the repository, so the finder walks directories and reads labels off
// the tree. A plan is held by the control plane and named by a tag, so there is
// nothing to walk: the repository contributes a list of tag globs, and
// selection is matching those against the tags that exist.
type PlanTagSelector struct {
	patterns    []string
	withTags    []string
	withoutTags []string
	repo        *GitRepo
}

// NewPlanTagSelector builds a selector from the repository's configuration.
//
// Explicit patterns take the place of the configured list, the way -d and -f
// do for smart tests, and mean the caller does not need to be inside a
// configured repository at all.
func NewPlanTagSelector(patterns, withTags, withoutTags []string) (*PlanTagSelector, error) {
	sel := &PlanTagSelector{
		patterns:    patterns,
		withTags:    withTags,
		withoutTags: withoutTags,
	}

	// A git repo is useful context to report even when the tags came from
	// flags, so look for one either way and do not fail if there is none.
	if cwd, err := os.Getwd(); err == nil {
		sel.repo, _ = FindGitRepo(cwd)
	}

	if len(sel.patterns) > 0 {
		return sel, nil
	}

	if sel.repo == nil {
		return nil, fmt.Errorf("not inside a git repository, and no plan tags were given")
	}
	cfg, err := LoadConfig(sel.repo)
	if err != nil {
		return nil, fmt.Errorf("failed to load .signadot/config.yaml: %w", err)
	}
	if len(cfg.Plans) == 0 {
		return nil, fmt.Errorf("plans is required in .signadot/config.yaml to run plans by tag")
	}
	sel.patterns = cfg.Plans
	return sel, nil
}

// GetGitRepo returns the repository the selection came from, or nil.
func (s *PlanTagSelector) GetGitRepo() *GitRepo {
	return s.repo
}

// Patterns returns the globs this selector matches against, which is worth
// reporting when nothing matched: the tags that do exist are of less use to
// the reader than what was being looked for.
func (s *PlanTagSelector) Patterns() []string {
	return s.patterns
}

// Select returns the subset of tagNames this repository selects, sorted, with
// duplicates removed.
//
// A tag is selected when it matches any configured pattern, matches every
// --with-tag, and matches no --without-tag. The asymmetry is deliberate and
// matches label filtering for smart tests: patterns widen the selection,
// with-filters narrow it, without-filters remove from it.
func (s *PlanTagSelector) Select(tagNames []string) ([]string, error) {
	seen := make(map[string]bool, len(tagNames))
	var selected []string

	for _, name := range tagNames {
		if seen[name] {
			continue
		}
		ok, err := s.matches(name)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		seen[name] = true
		selected = append(selected, name)
	}

	sort.Strings(selected)
	return selected, nil
}

func (s *PlanTagSelector) matches(name string) (bool, error) {
	ok, err := matchesAny(s.patterns, name)
	if err != nil || !ok {
		return false, err
	}
	for _, pattern := range s.withTags {
		ok, err := filepath.Match(pattern, name)
		if err != nil {
			return false, fmt.Errorf("invalid --with-tag pattern %q: %w", pattern, err)
		}
		if !ok {
			return false, nil
		}
	}
	excluded, err := matchesAny(s.withoutTags, name)
	if err != nil {
		return false, err
	}
	return !excluded, nil
}

func matchesAny(patterns []string, name string) (bool, error) {
	for _, pattern := range patterns {
		ok, err := filepath.Match(pattern, name)
		if err != nil {
			return false, fmt.Errorf("invalid tag pattern %q: %w", pattern, err)
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}
