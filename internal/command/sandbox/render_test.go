package sandbox

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/signadot/cli/internal/config"
)

var update = flag.Bool("update", false, "rewrite the golden files from the current output")

const fixturesDir = "testdata/render"

// fixtureOptions overrides the flag defaults for a fixture.
type fixtureOptions struct {
	CIContext     string `json:"ciContext"`
	DefaultLabels *bool  `json:"defaultLabels"`
	Name          string `json:"name"`
	TTL           string `json:"ttl"`
	// Set contains --set arguments, as var=val.
	Set []string `json:"set"`
	// Error, when set, is a substring the fixture is expected to fail with.
	Error string `json:"error"`
}

// Each fixture is a document, plus the environment and flags it renders under,
// and the spec that --dry-run=client should produce. Together they cover the
// whole path: routing between the two kinds of -f document, compiling values,
// rendering the built-in template, patching, and the strict decode at the end.
//
// They are deliberately plain YAML, because their job is to make a change in the
// rendered output visible in review.
func TestRenderGolden(t *testing.T) {
	entries, err := os.ReadDir(fixturesDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			runFixture(t, filepath.Join(fixturesDir, e.Name()))
		})
	}
}

func runFixture(t *testing.T, dir string) {
	t.Helper()

	opts := fixtureOptions{CIContext: "auto"}
	readJSON(t, filepath.Join(dir, "options.json"), &opts)

	env := map[string]string{}
	readJSON(t, filepath.Join(dir, "env.json"), &env)
	// The rendering path reads the process environment, as it does in a real
	// run, so the fixture's environment has to be installed rather than injected.
	for k, v := range env {
		t.Setenv(k, v)
	}
	for _, k := range []string{"GITHUB_ACTIONS", "GITHUB_REPOSITORY", "GITHUB_REF",
		"GITHUB_SHA", "GITHUB_HEAD_REF", "GITHUB_EVENT_PATH"} {
		if _, ok := env[k]; !ok {
			t.Setenv(k, "")
		}
	}

	cfg := &config.SandboxApply{
		Sandbox:       &config.Sandbox{API: &config.API{}},
		Filename:      inputFile(t, dir),
		DryRun:        config.DryRunClient,
		CIContext:     opts.CIContext,
		DefaultLabels: true,
		Name:          opts.Name,
		TTL:           opts.TTL,
	}
	if opts.DefaultLabels != nil {
		cfg.DefaultLabels = *opts.DefaultLabels
	}
	if patch := filepath.Join(dir, "patch.yaml"); exists(patch) {
		cfg.Patch = patch
	}
	for _, s := range opts.Set {
		if err := cfg.TemplateVals.Set(s); err != nil {
			t.Fatal(err)
		}
	}

	doc, _, err := renderSandbox(cfg)
	if opts.Error != "" {
		if err == nil || !strings.Contains(err.Error(), opts.Error) {
			t.Fatalf("expected an error containing %q, got %v", opts.Error, err)
		}
		return
	}
	if err != nil {
		t.Fatalf("rendering: %v", err)
	}

	var got bytes.Buffer
	if err := writeRenderedSpec(cfg, &got, doc); err != nil {
		t.Fatal(err)
	}

	goldenPath := filepath.Join(dir, "expected.yaml")
	if *update {
		if err := os.WriteFile(goldenPath, got.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != string(want) {
		t.Errorf("rendered spec does not match %s\n--- got ---\n%s\n--- want ---\n%s",
			goldenPath, got.String(), want)
	}
}

// A rendered spec must render to itself. The Action depends on this: it renders
// with --dry-run=client, publishes the result, and then applies that same file,
// which is only equivalent to applying directly if rendering is a fixed point.
func TestRenderedSpecIsAFixedPoint(t *testing.T) {
	entries, err := os.ReadDir(fixturesDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		golden := filepath.Join(fixturesDir, e.Name(), "expected.yaml")
		if !e.IsDir() || !exists(golden) {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			cfg := &config.SandboxApply{
				Sandbox:  &config.Sandbox{API: &config.API{}},
				Filename: golden,
				DryRun:   config.DryRunClient,
				// No context, so nothing is added on the second pass that was
				// not already in the file.
				CIContext:     "none",
				DefaultLabels: true,
			}
			doc, _, err := renderSandbox(cfg)
			if err != nil {
				t.Fatalf("re-rendering %s: %v", golden, err)
			}
			var got bytes.Buffer
			if err := writeRenderedSpec(cfg, &got, doc); err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatal(err)
			}
			if got.String() != string(want) {
				t.Errorf("re-rendering changed the spec\n--- got ---\n%s\n--- want ---\n%s",
					got.String(), want)
			}
		})
	}
}

// inputFile is the fixture's -f argument: a values document, or a sandbox spec
// for the fixtures that exercise that route.
func inputFile(t *testing.T, dir string) string {
	t.Helper()
	for _, name := range []string{"values.yaml", "spec.yaml"} {
		if p := filepath.Join(dir, name); exists(p) {
			return p
		}
	}
	t.Fatalf("%s has neither values.yaml nor spec.yaml", dir)
	return ""
}

func readJSON(t *testing.T, path string, into any) {
	t.Helper()
	d, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(d, into); err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
