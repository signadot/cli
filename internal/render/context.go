package render

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/signadot/go-sdk/models"
)

const (
	UsageLabelKey     = "signadot/usage"
	LabelGitHubRepo   = "signadot/github-repo"
	LabelGitHubPR     = "signadot/github-pull-request"
	LabelGitHubBranch = "signadot/github-branch"

	// MaxNameLen is the longest sandbox name the Signadot API accepts.
	MaxNameLen = 30
)

// CIContext describes the CI run a sandbox is being created from. It is what
// lets a pipeline omit the sandbox name entirely and still converge on the same
// sandbox each time it runs for a given pull request.
type CIContext struct {
	// Provider is the CI system this context was derived from, or "" when none
	// was detected.
	Provider string
	Detected bool
	// Repo is the fully qualified repository, e.g. "signadot/hotrod".
	Repo string
	// RepoSlug is the slugified trailing segment of Repo, e.g. "hotrod".
	RepoSlug   string
	PR         string
	SHA        string
	ShortSHA   string
	BranchSlug string
}

// Env is the environment a context is detected from. Taking it as a parameter
// rather than reading the process environment keeps rendering testable.
type Env func(string) string

// OSEnv reads the process environment.
func OSEnv(k string) string { return os.Getenv(k) }

// MapEnv reads a fixed map, for tests.
func MapEnv(m map[string]string) Env {
	return func(k string) string { return m[k] }
}

func NoContext() CIContext { return CIContext{} }

func DetectGitHub(env Env) CIContext {
	repo := env("GITHUB_REPOSITORY")
	sha := env("GITHUB_SHA")
	return CIContext{
		Provider:   "github",
		Detected:   true,
		Repo:       repo,
		RepoSlug:   Slugify(lastPathSegment(repo)),
		PR:         githubPRNumber(env),
		SHA:        sha,
		ShortSHA:   shortSHA(sha),
		BranchSlug: Slugify(env("GITHUB_HEAD_REF")),
	}
}

// Detect resolves the --ci-context flag: "auto" uses the provider's own marker
// variable to decide, a named provider forces detection, "none" disables it.
func Detect(mode string, env Env) (CIContext, error) {
	switch mode {
	case "", "auto":
		if env("GITHUB_ACTIONS") == "true" {
			return DetectGitHub(env), nil
		}
		return NoContext(), nil
	case "github":
		return DetectGitHub(env), nil
	case "none":
		return NoContext(), nil
	default:
		return CIContext{}, fmt.Errorf("unknown CI context %q: expected one of auto, github, none", mode)
	}
}

// ProviderLabels are the built-in labels that let the Signadot App correlate a
// sandbox back to the pull request that produced it, and delete it when that
// pull request closes.
func ProviderLabels(c CIContext) map[string]string {
	if c.Provider != "github" {
		return nil
	}
	return map[string]string{
		LabelGitHubRepo: c.Repo,
		LabelGitHubPR:   c.PR,
	}
}

// DefaultName derives a stable sandbox name from the CI context so that repeated
// runs on the same pull request update one sandbox instead of accumulating them.
func DefaultName(c CIContext) string {
	switch {
	case c.RepoSlug == "":
		return ""
	case c.PR != "":
		return fmt.Sprintf("%s-pr-%s", c.RepoSlug, c.PR)
	case c.ShortSHA != "":
		return fmt.Sprintf("%s-%s", c.RepoSlug, c.ShortSHA)
	default:
		return c.RepoSlug
	}
}

// ImagePlaceholders are the values available to defaults.imageTemplate.
func ImagePlaceholders(c CIContext, workload, namespace string) map[string]string {
	return map[string]string{
		"workload":    workload,
		"namespace":   namespace,
		"sha":         c.SHA,
		"short-sha":   c.ShortSHA,
		"pr":          c.PR,
		"branch-slug": c.BranchSlug,
	}
}

// ResolveName picks the sandbox name: --name wins, then the name in the
// document, then a name derived from the CI context.
//
// Only names we synthesise, or that arrive by flag from a CI variable, are
// normalised. A name written in a document is left exactly as authored, so that
// adopting this path cannot silently retarget an existing sandbox.
func ResolveName(docName, explicitName string, c CIContext) (string, error) {
	switch {
	case explicitName != "":
		return NormalizeName(explicitName), nil
	case docName != "":
		return docName, nil
	case c.Detected && DefaultName(c) != "":
		return NormalizeName(DefaultName(c)), nil
	default:
		return "", fmt.Errorf("sandbox name is required: set --name, the document's name, " +
			"or run in a detected CI context")
	}
}

// ApplyContext resolves the sandbox name and TTL and stamps the built-in labels.
// Everything it sets is a default: an explicit value from the flags or the
// document always wins.
func ApplyContext(sb *models.Sandbox, c CIContext, explicitName, explicitTTL string, defaultLabels bool) error {
	name, err := ResolveName(sb.Name, explicitName, c)
	if err != nil {
		return err
	}
	sb.Name = name

	if sb.Spec == nil {
		sb.Spec = &models.SandboxSpec{}
	}

	// No implicit TTL. A sandbox outlives the job that made it by design, and
	// picking an expiry on the caller's behalf would quietly delete work someone
	// is still reviewing.
	if explicitTTL != "" {
		sb.Spec.TTL = &models.SandboxTTL{Duration: explicitTTL}
	}

	if !defaultLabels || !c.Detected {
		return nil
	}

	labels := sb.Spec.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	if _, ok := labels[UsageLabelKey]; !ok {
		labels[UsageLabelKey] = "ci"
	}
	for k, v := range ProviderLabels(c) {
		if v == "" {
			continue
		}
		if _, ok := labels[k]; !ok {
			labels[k] = v
		}
	}
	sb.Spec.Labels = labels
	return nil
}

// NormalizeName slugifies a candidate name and, when it is too long for the API,
// truncates it and appends a content hash so distinct inputs stay distinct.
func NormalizeName(name string) string {
	slug := Slugify(name)
	if len(slug) <= MaxNameLen {
		return slug
	}
	sum := sha256.Sum256([]byte(slug))
	suffix := hex.EncodeToString(sum[:])[:6]
	keep := MaxNameLen - 1 - len(suffix)
	if keep < 1 {
		keep = 1
	}
	if keep > len(slug) {
		keep = len(slug)
	}
	return strings.TrimRight(slug[:keep], "-") + "-" + suffix
}

var nonAlnumRx = regexp.MustCompile(`[^a-z0-9]+`)

func Slugify(s string) string {
	return strings.Trim(nonAlnumRx.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

func shortSHA(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}

func lastPathSegment(s string) string {
	if i := strings.LastIndexByte(s, '/'); i >= 0 {
		return s[i+1:]
	}
	return s
}

var prRefRx = regexp.MustCompile(`^refs/pull/(\d+)/`)

// githubPRNumber reads the PR number from GITHUB_REF when available, falling
// back to the event payload for triggers such as pull_request_target and
// issue_comment where the ref does not carry it.
func githubPRNumber(env Env) string {
	if m := prRefRx.FindStringSubmatch(env("GITHUB_REF")); m != nil {
		return m[1]
	}

	eventPath := env("GITHUB_EVENT_PATH")
	if eventPath == "" {
		return ""
	}
	d, err := os.ReadFile(eventPath)
	if err != nil {
		return ""
	}
	var event struct {
		PullRequest *struct {
			Number *int64 `json:"number"`
		} `json:"pull_request"`
		Number *int64 `json:"number"`
	}
	if err := json.Unmarshal(d, &event); err != nil {
		// A malformed payload just means we cannot derive a PR number.
		return ""
	}
	if event.PullRequest != nil && event.PullRequest.Number != nil && *event.PullRequest.Number != 0 {
		return fmt.Sprint(*event.PullRequest.Number)
	}
	if event.Number != nil && *event.Number != 0 {
		return fmt.Sprint(*event.Number)
	}
	return ""
}
