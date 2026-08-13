package render

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/signadot/go-sdk/models"
)

func sandboxWithCluster(cluster string) *models.Sandbox {
	return &models.Sandbox{Spec: &models.SandboxSpec{Cluster: &cluster}}
}

func TestSlugify(t *testing.T) {
	tests := map[string]string{
		"hotrod":               "hotrod",
		"My/Explicit-Sandbox":  "my-explicit-sandbox",
		"feature/new-thing":    "feature-new-thing",
		"--leading-trailing--": "leading-trailing",
		"UPPER_case 123":       "upper-case-123",
		"":                     "",
	}
	for in, want := range tests {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeName(t *testing.T) {
	// Short names pass through slugified.
	if got := NormalizeName("Route PR 12"); got != "route-pr-12" {
		t.Errorf("got %q", got)
	}

	// Long names are truncated and hash-suffixed, and stay within the limit.
	long := "platform-payments-authorization-service-pr-9876"
	got := NormalizeName(long)
	if len(got) > MaxNameLen {
		t.Errorf("normalized name %q is %d chars, over the %d limit", got, len(got), MaxNameLen)
	}
	if got != "platform-payments-autho-9a5257" {
		t.Errorf("got %q", got)
	}

	// The hash keeps distinct inputs distinct where truncation alone would not.
	a := NormalizeName(long + "-alpha")
	b := NormalizeName(long + "-bravo")
	if a == b {
		t.Errorf("distinct long names collided on %q", a)
	}

	// Truncation must not leave a trailing dash.
	if n := NormalizeName("aaaaaaaaaaaaaaaaaaaaaaa-bbbbbbbbb"); n[MaxNameLen-8] == '-' {
		t.Errorf("got a dash before the hash suffix: %q", n)
	}

	// Normalizing is idempotent, so a name can pass through more than once.
	if n := NormalizeName(got); n != got {
		t.Errorf("NormalizeName(%q) = %q", got, n)
	}
}

func TestDetect(t *testing.T) {
	gh := map[string]string{
		"GITHUB_ACTIONS":    "true",
		"GITHUB_REPOSITORY": "signadot/hotrod",
		"GITHUB_REF":        "refs/pull/345/merge",
		"GITHUB_SHA":        "0123456789abcdef0123456789abcdef01234567",
		"GITHUB_HEAD_REF":   "joe/multi-fork",
	}

	c, err := Detect("auto", MapEnv(gh))
	if err != nil {
		t.Fatal(err)
	}
	if !c.Detected || c.Provider != "github" {
		t.Fatalf("expected GitHub detection, got %+v", c)
	}
	if c.RepoSlug != "hotrod" || c.PR != "345" || c.ShortSHA != "0123456" ||
		c.BranchSlug != "joe-multi-fork" {
		t.Errorf("unexpected context %+v", c)
	}

	// auto outside a runner detects nothing.
	c, err = Detect("auto", MapEnv(map[string]string{}))
	if err != nil {
		t.Fatal(err)
	}
	if c.Detected {
		t.Errorf("expected no detection, got %+v", c)
	}

	// none disables detection even inside a runner.
	c, _ = Detect("none", MapEnv(gh))
	if c.Detected {
		t.Errorf("context none should not detect, got %+v", c)
	}

	// github forces detection outside a runner.
	c, _ = Detect("github", MapEnv(map[string]string{"GITHUB_REPOSITORY": "a/b"}))
	if !c.Detected || c.RepoSlug != "b" {
		t.Errorf("unexpected context %+v", c)
	}

	if _, err := Detect("gitlab", MapEnv(gh)); err == nil {
		t.Error("expected an error for an unknown CI context")
	}
}

// pull_request_target and issue_comment do not carry the PR number in the ref,
// so it has to come from the event payload.
func TestPRNumberFromEventPayload(t *testing.T) {
	dir := t.TempDir()

	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	cases := []struct {
		name string
		body string
		want string
	}{
		{"pull_request.json", `{"pull_request":{"number":42}}`, "42"},
		{"issue_comment.json", `{"number":7}`, "7"},
		{"unrelated.json", `{"ref":"refs/heads/main"}`, ""},
		{"malformed.json", `{not json`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := DetectGitHub(MapEnv(map[string]string{
				"GITHUB_REPOSITORY": "acme/route",
				"GITHUB_EVENT_PATH": write(tc.name, tc.body),
			}))
			if c.PR != tc.want {
				t.Errorf("got PR %q, want %q", c.PR, tc.want)
			}
		})
	}

	// The ref wins when it carries a number, without reading the payload.
	c := DetectGitHub(MapEnv(map[string]string{
		"GITHUB_REF":        "refs/pull/99/merge",
		"GITHUB_EVENT_PATH": write("both.json", `{"pull_request":{"number":42}}`),
	}))
	if c.PR != "99" {
		t.Errorf("got PR %q, want 99", c.PR)
	}
}

func TestDefaultName(t *testing.T) {
	tests := []struct {
		ctx  CIContext
		want string
	}{
		{CIContext{RepoSlug: "hotrod", PR: "12"}, "hotrod-pr-12"},
		{CIContext{RepoSlug: "hotrod", ShortSHA: "abc1234"}, "hotrod-abc1234"},
		{CIContext{RepoSlug: "hotrod"}, "hotrod"},
		{CIContext{}, ""},
	}
	for _, tc := range tests {
		if got := DefaultName(tc.ctx); got != tc.want {
			t.Errorf("DefaultName(%+v) = %q, want %q", tc.ctx, got, tc.want)
		}
	}
}

func TestResolveName(t *testing.T) {
	ctx := DetectGitHub(MapEnv(map[string]string{
		"GITHUB_REPOSITORY": "acme/route",
		"GITHUB_REF":        "refs/pull/12/merge",
	}))

	// A name from the flag is normalized, since it usually comes from a CI
	// variable such as a branch name.
	if got, _ := ResolveName("", "Feature/Fix_It", ctx); got != "feature-fix-it" {
		t.Errorf("got %q", got)
	}
	// The flag beats the document.
	if got, _ := ResolveName("in-doc", "from-flag", ctx); got != "from-flag" {
		t.Errorf("got %q", got)
	}
	// A name written in a document is left exactly as authored, so adopting this
	// path cannot silently retarget an existing sandbox.
	if got, _ := ResolveName("Keep_As_Is", "", ctx); got != "Keep_As_Is" {
		t.Errorf("got %q", got)
	}
	// Otherwise the CI context supplies one.
	if got, _ := ResolveName("", "", ctx); got != "route-pr-12" {
		t.Errorf("got %q", got)
	}
	if _, err := ResolveName("", "", NoContext()); err == nil {
		t.Error("expected an error with no name and no context")
	}
}

// Everything ApplyContext sets is a default; explicit values always win.
func TestApplyContextDefersToExplicitValues(t *testing.T) {
	ctx := DetectGitHub(MapEnv(map[string]string{
		"GITHUB_REPOSITORY": "acme/route",
		"GITHUB_REF":        "refs/pull/12/merge",
	}))

	t.Run("derives name and labels, but no ttl", func(t *testing.T) {
		sb := sandboxWithCluster("c")
		if err := ApplyContext(sb, ctx, "", "", true); err != nil {
			t.Fatal(err)
		}
		if sb.Name != "route-pr-12" {
			t.Errorf("got name %q", sb.Name)
		}
		// Lifetime belongs to the GitHub App, not to a default invented here.
		if sb.Spec.TTL != nil {
			t.Errorf("did not expect a ttl, got %+v", sb.Spec.TTL)
		}
		want := map[string]string{
			UsageLabelKey:   "ci",
			LabelGitHubRepo: "acme/route",
			LabelGitHubPR:   "12",
		}
		for k, v := range want {
			if sb.Spec.Labels[k] != v {
				t.Errorf("label %s = %q, want %q", k, sb.Spec.Labels[k], v)
			}
		}
	})

	t.Run("keeps values already set", func(t *testing.T) {
		sb := sandboxWithCluster("c")
		sb.Name = "mine"
		sb.Spec.TTL = &models.SandboxTTL{Duration: "30m"}
		sb.Spec.Labels = map[string]string{UsageLabelKey: "manual"}
		if err := ApplyContext(sb, ctx, "", "", true); err != nil {
			t.Fatal(err)
		}
		if sb.Name != "mine" {
			t.Errorf("got name %q", sb.Name)
		}
		if sb.Spec.TTL.Duration != "30m" {
			t.Errorf("got ttl %q", sb.Spec.TTL.Duration)
		}
		if sb.Spec.Labels[UsageLabelKey] != "manual" {
			t.Errorf("got usage label %q", sb.Spec.Labels[UsageLabelKey])
		}
		// The correlation labels are still added alongside.
		if sb.Spec.Labels[LabelGitHubPR] != "12" {
			t.Errorf("expected the PR label to be added")
		}
	})

	t.Run("an explicit ttl overrides the document", func(t *testing.T) {
		sb := sandboxWithCluster("c")
		sb.Spec.TTL = &models.SandboxTTL{Duration: "30m"}
		if err := ApplyContext(sb, ctx, "", "4h", true); err != nil {
			t.Fatal(err)
		}
		if sb.Spec.TTL.Duration != "4h" {
			t.Errorf("got ttl %q", sb.Spec.TTL.Duration)
		}
	})

	t.Run("default-labels off adds nothing", func(t *testing.T) {
		sb := sandboxWithCluster("c")
		if err := ApplyContext(sb, ctx, "", "", false); err != nil {
			t.Fatal(err)
		}
		if len(sb.Spec.Labels) != 0 {
			t.Errorf("got labels %+v", sb.Spec.Labels)
		}
	})

	t.Run("no context and no name is an error", func(t *testing.T) {
		sb := sandboxWithCluster("c")
		if err := ApplyContext(sb, NoContext(), "", "", true); err == nil {
			t.Error("expected an error")
		}
	})

	t.Run("no context means no labels and no default ttl", func(t *testing.T) {
		sb := sandboxWithCluster("c")
		if err := ApplyContext(sb, NoContext(), "explicit", "", true); err != nil {
			t.Fatal(err)
		}
		if sb.Spec.TTL != nil {
			t.Errorf("got ttl %+v", sb.Spec.TTL)
		}
		if len(sb.Spec.Labels) != 0 {
			t.Errorf("got labels %+v", sb.Spec.Labels)
		}
	})
}

// A sandbox created outside a pull request must not carry an empty PR label,
// which would otherwise confuse the App integration.
func TestProviderLabelsSkipEmptyPR(t *testing.T) {
	ctx := DetectGitHub(MapEnv(map[string]string{
		"GITHUB_REPOSITORY": "acme/route",
		"GITHUB_SHA":        "abc1234def",
	}))
	sb := sandboxWithCluster("c")
	if err := ApplyContext(sb, ctx, "", "", true); err != nil {
		t.Fatal(err)
	}
	if _, ok := sb.Spec.Labels[LabelGitHubPR]; ok {
		t.Errorf("expected no PR label, got %+v", sb.Spec.Labels)
	}
	if sb.Spec.Labels[LabelGitHubRepo] != "acme/route" {
		t.Errorf("got labels %+v", sb.Spec.Labels)
	}
}

func TestEnvValueUnmarshal(t *testing.T) {
	var got map[string]EnvValue
	in := `{"A":"str","B":{"fromResource":"db.host"},"C":8080,"D":true,"E":null}`
	if err := json.Unmarshal([]byte(in), &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]EnvValue{
		"A": {Value: "str"},
		"B": {FromResource: "db.host", IsRef: true},
		"C": {Value: "8080"},
		"D": {Value: "true"},
		"E": {Value: ""},
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("%s: got %+v, want %+v", k, got[k], w)
		}
	}

	for _, bad := range []string{`{"A":["list"]}`, `{"A":{"other":"x"}}`} {
		var m map[string]EnvValue
		if err := json.Unmarshal([]byte(bad), &m); err == nil {
			t.Errorf("%s: expected an error", bad)
		}
	}
}
