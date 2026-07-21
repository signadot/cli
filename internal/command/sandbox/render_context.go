package sandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/signadot/go-sdk/models"
)

const (
	contextAuto     = "auto"
	contextNone     = "none"
	contextGitHub   = "github"
	contextGitLab   = "gitlab"
	contextCircleCI = "circleci"

	// defaultCITTL is the safety-net TTL stamped on CI sandboxes when none is
	// otherwise specified (§2.3).
	defaultCITTL = "2d"

	// maxNameLen is the sandbox name byte limit (§2.3).
	maxNameLen = 30

	// GitHub lifecycle label keys read by the Signadot GitHub App for PR
	// comments and delete-on-close. These must match the App's contract
	// exactly so the integration works with zero configuration:
	// https://www.signadot.com/docs/guides/integrate-ci/github#specification
	labelGitHubRepo = "signadot/github-repo"
	labelGitHubPR   = "signadot/github-pull-request"
)

// CIContext holds the CI provider facts used for deterministic naming, image
// template placeholders and provider labels (§2.3).
type CIContext struct {
	Provider   string
	Repo       string // owner/repo (github), group/project (gitlab)
	RepoSlug   string // slugified last path segment of Repo
	PR         string // PR / MR number
	SHA        string
	ShortSHA   string
	BranchSlug string
	Detected   bool
}

// detectCIContext resolves the CI context according to --context mode.
func detectCIContext(mode string) (*CIContext, error) {
	switch mode {
	case contextNone:
		return &CIContext{Provider: contextNone}, nil
	case contextGitHub:
		return detectGitHub(), nil
	case contextGitLab:
		return detectGitLab(), nil
	case contextCircleCI:
		return detectCircleCI(), nil
	case contextAuto, "":
		switch {
		case os.Getenv("GITHUB_ACTIONS") == "true":
			return detectGitHub(), nil
		case os.Getenv("GITLAB_CI") == "true":
			return detectGitLab(), nil
		case os.Getenv("CIRCLECI") == "true":
			return detectCircleCI(), nil
		default:
			return &CIContext{Provider: contextNone}, nil
		}
	default:
		return nil, fmt.Errorf("unknown --context %q (want auto|none|github|gitlab|circleci)", mode)
	}
}

func detectGitHub() *CIContext {
	c := &CIContext{Provider: contextGitHub, Detected: true}
	c.Repo = os.Getenv("GITHUB_REPOSITORY")
	c.RepoSlug = slugify(lastPathSegment(c.Repo))
	c.SHA = os.Getenv("GITHUB_SHA")
	c.ShortSHA = shortSHA(c.SHA)
	c.BranchSlug = slugify(os.Getenv("GITHUB_HEAD_REF"))
	c.PR = githubPRNumber()
	return c
}

// detectGitLab is a scaffold: detection works, but MR-lifecycle label parity
// depends on GitLab integration support (design §7).
func detectGitLab() *CIContext {
	c := &CIContext{Provider: contextGitLab, Detected: true}
	c.Repo = os.Getenv("CI_PROJECT_PATH")
	c.RepoSlug = slugify(lastPathSegment(c.Repo))
	c.SHA = os.Getenv("CI_COMMIT_SHA")
	c.ShortSHA = shortSHA(c.SHA)
	c.BranchSlug = slugify(os.Getenv("CI_COMMIT_REF_SLUG"))
	c.PR = os.Getenv("CI_MERGE_REQUEST_IID")
	return c
}

// detectCircleCI is a scaffold (design §8).
func detectCircleCI() *CIContext {
	c := &CIContext{Provider: contextCircleCI, Detected: true}
	c.Repo = os.Getenv("CIRCLE_PROJECT_REPONAME")
	c.RepoSlug = slugify(lastPathSegment(c.Repo))
	c.SHA = os.Getenv("CIRCLE_SHA1")
	c.ShortSHA = shortSHA(c.SHA)
	c.BranchSlug = slugify(os.Getenv("CIRCLE_BRANCH"))
	c.PR = circleCIPRNumber()
	return c
}

func (c *CIContext) imagePlaceholders(workload, namespace string) map[string]string {
	return map[string]string{
		"workload":    workload,
		"namespace":   namespace,
		"sha":         c.SHA,
		"short-sha":   c.ShortSHA,
		"pr":          c.PR,
		"branch-slug": c.BranchSlug,
	}
}

// providerLabels returns the lifecycle labels the Signadot App/integration uses
// for PR comments and delete-on-close (§2.3).
func (c *CIContext) providerLabels() map[string]string {
	switch c.Provider {
	case contextGitHub:
		return map[string]string{labelGitHubRepo: c.Repo, labelGitHubPR: c.PR}
	case contextGitLab:
		// TODO: confirm the App's label keys once GitLab integration support
		// lands (§7); likely signadot/gitlab-* following the GitHub pattern.
		return map[string]string{"signadot/gitlab-project": c.Repo, "signadot/gitlab-merge-request": c.PR}
	case contextCircleCI:
		// TODO: confirm the App's label keys for CircleCI (§8).
		return map[string]string{"signadot/circleci-repo": c.Repo, "signadot/circleci-pull-request": c.PR}
	}
	return nil
}

func (c *CIContext) defaultName() string {
	if c.RepoSlug == "" {
		return ""
	}
	if c.PR != "" {
		return fmt.Sprintf("%s-pr-%s", c.RepoSlug, c.PR)
	}
	if c.ShortSHA != "" {
		return fmt.Sprintf("%s-%s", c.RepoSlug, c.ShortSHA)
	}
	return c.RepoSlug
}

// applyContext stamps CI-derived name, labels and TTL onto the rendered sandbox.
// explicitName/explicitTTL come from flags and take precedence over everything.
func applyContext(sb *models.Sandbox, c *CIContext, explicitName, explicitTTL string) error {
	// Resolve name precedence: flag > values.name > deterministic default.
	name := explicitName
	if name == "" {
		name = sb.Name
	}
	if name == "" && c != nil && c.Detected {
		name = c.defaultName()
	}
	if name == "" {
		return fmt.Errorf("sandbox name is required: set --name, values.name, or run in a detected CI context")
	}
	sb.Name = normalizeName(name)

	// TTL precedence: flag > values.ttl > CI default.
	switch {
	case explicitTTL != "":
		sb.Spec.TTL = &models.SandboxTTL{Duration: explicitTTL}
	case sb.Spec.TTL != nil:
		// keep values.ttl
	case c != nil && c.Detected:
		sb.Spec.TTL = &models.SandboxTTL{Duration: defaultCITTL}
	}

	// Labels: only when a CI provider was detected.
	if c == nil || !c.Detected {
		return nil
	}
	if sb.Spec.Labels == nil {
		sb.Spec.Labels = map[string]string{}
	}
	if _, ok := sb.Spec.Labels[usageLabelKey]; !ok {
		sb.Spec.Labels[usageLabelKey] = "ci"
	}
	for k, v := range c.providerLabels() {
		if v == "" {
			continue
		}
		if _, ok := sb.Spec.Labels[k]; !ok {
			sb.Spec.Labels[k] = v
		}
	}
	return nil
}

var (
	nonSlugRx      = regexp.MustCompile(`[^a-z0-9]+`)
	githubPRRefRx  = regexp.MustCompile(`^refs/pull/(\d+)/`)
	circlePRURLRx  = regexp.MustCompile(`/(\d+)$`)
)

func slugify(s string) string {
	s = strings.ToLower(s)
	s = nonSlugRx.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// normalizeName slugifies and enforces the 30-byte limit, appending a 6-char
// hash of the original when truncation is required (§2.3).
func normalizeName(name string) string {
	slug := slugify(name)
	if len(slug) <= maxNameLen {
		return slug
	}
	h := sha256.Sum256([]byte(slug))
	suffix := hex.EncodeToString(h[:])[:6]
	keep := maxNameLen - 1 - len(suffix)
	if keep < 1 {
		keep = 1
	}
	return strings.TrimRight(slug[:keep], "-") + "-" + suffix
}

func lastPathSegment(s string) string {
	if i := strings.LastIndex(s, "/"); i >= 0 {
		return s[i+1:]
	}
	return s
}

func shortSHA(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}

func githubPRNumber() string {
	if m := githubPRRefRx.FindStringSubmatch(os.Getenv("GITHUB_REF")); m != nil {
		return m[1]
	}
	if path := os.Getenv("GITHUB_EVENT_PATH"); path != "" {
		if data, err := os.ReadFile(path); err == nil {
			var ev struct {
				Number      int `json:"number"`
				PullRequest struct {
					Number int `json:"number"`
				} `json:"pull_request"`
			}
			if json.Unmarshal(data, &ev) == nil {
				if ev.PullRequest.Number != 0 {
					return strconv.Itoa(ev.PullRequest.Number)
				}
				if ev.Number != 0 {
					return strconv.Itoa(ev.Number)
				}
			}
		}
	}
	return ""
}

func circleCIPRNumber() string {
	if n := os.Getenv("CIRCLE_PR_NUMBER"); n != "" {
		return n
	}
	if m := circlePRURLRx.FindStringSubmatch(os.Getenv("CIRCLE_PULL_REQUEST")); m != nil {
		return m[1]
	}
	return ""
}
