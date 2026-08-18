package sandbox

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/signadot/cli/internal/config"
)

const dryRunTemplate = `name: test-sandbox
spec:
  cluster: my-cluster
  description: build @{tag}
  forks:
    - forkOf:
        kind: Deployment
        name: route
        namespace: hotrod
      customizations:
        images:
          - image: acme/route:@{tag}
`

// dryRunConfig builds an apply config with no credentials of any kind, so that a
// test reaching authentication fails rather than silently using the developer's
// own login.
func dryRunConfig(t *testing.T, file string, mode config.DryRunMode, sets ...string) *config.SandboxApply {
	t.Helper()
	cfg := &config.SandboxApply{
		Sandbox:  &config.Sandbox{API: &config.API{}},
		Filename: file,
		DryRun:   mode,
	}
	for _, s := range sets {
		if err := cfg.TemplateVals.Set(s); err != nil {
			t.Fatalf("--set %s: %v", s, err)
		}
	}
	return cfg
}

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func render(t *testing.T, file string, sets ...string) string {
	t.Helper()
	var out, log bytes.Buffer
	cfg := dryRunConfig(t, file, config.DryRunClient, sets...)
	if err := apply(cfg, &out, &log, nil); err != nil {
		t.Fatalf("--dry-run=client: %v", err)
	}
	return out.String()
}

// The point of --dry-run=client is that it renders and validates without
// contacting anything, so it has to work for a caller who has never logged in.
func TestDryRunClientNeedsNoCredentials(t *testing.T) {
	got := render(t, writeTemp(t, "sandbox.yaml", dryRunTemplate), "tag=abc123")

	for _, want := range []string{"name: test-sandbox", "acme/route:abc123", "build abc123"} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered spec is missing %q:\n%s", want, got)
		}
	}

	// InitAPIConfig is what builds the client, so an unset client is the evidence
	// that rendering returned before authentication was even attempted.
	cfg := dryRunConfig(t, writeTemp(t, "sandbox.yaml", dryRunTemplate), config.DryRunClient)
	if err := cfg.TemplateVals.Set("tag=abc123"); err != nil {
		t.Fatal(err)
	}
	var out, log bytes.Buffer
	if err := apply(cfg, &out, &log, nil); err != nil {
		t.Fatal(err)
	}
	if cfg.Client != nil {
		t.Error("--dry-run=client initialised an API client")
	}
}

// The Action renders in one step and applies those exact bytes in another, which
// only holds if a rendered spec renders to itself.
func TestDryRunOutputIsAFixedPoint(t *testing.T) {
	once := render(t, writeTemp(t, "sandbox.yaml", dryRunTemplate), "tag=abc123")
	twice := render(t, writeTemp(t, "rendered.yaml", once))

	if once != twice {
		t.Errorf("re-rendering changed the spec:\nfirst:\n%s\nsecond:\n%s", once, twice)
	}
	if strings.Contains(once, "@{") {
		t.Errorf("rendered spec still carries a placeholder:\n%s", once)
	}
}

// Local validation is worth having only if it reports what the API would.
func TestDryRunClientValidates(t *testing.T) {
	for name, tc := range map[string]struct{ doc, want string }{
		"no cluster":    {"name: sb\nspec:\n  description: nothing\n", "cluster"},
		"unknown field": {"name: sb\nspec:\n  cluster: c\n  forkz: []\n", "unknown field"},
		"unset var":     {"name: sb-@{missing}\nspec:\n  cluster: c\n", "missing"},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := dryRunConfig(t, writeTemp(t, "sandbox.yaml", tc.doc), config.DryRunClient)
			var out, log bytes.Buffer
			err := apply(cfg, &out, &log, nil)
			if err == nil {
				t.Fatalf("expected an error, got output:\n%s", out.String())
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not mention %q", err, tc.want)
			}
		})
	}
}

// The mode exists in the flag's grammar so that adding it later needs no new
// spelling, but it has nothing behind it yet.
func TestDryRunServerIsRejected(t *testing.T) {
	cfg := dryRunConfig(t, writeTemp(t, "sandbox.yaml", dryRunTemplate), config.DryRunServer)
	var out, log bytes.Buffer
	err := apply(cfg, &out, &log, nil)
	if err == nil || !strings.Contains(err.Error(), "not available yet") {
		t.Errorf("got %v, want an error saying server dry run is not available yet", err)
	}
}
