package render

import (
	"strings"
	"testing"

	"github.com/signadot/go-sdk/models"
)

func baseValues() *Values {
	return &Values{
		Cluster:  "prod-eks",
		Defaults: &ValuesDefaults{Namespace: "hotrod"},
		Forks:    []ValuesFork{{Workload: "route", Image: "acme/route:1"}},
	}
}

func str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func TestCompileRequiresClusterAndForks(t *testing.T) {
	v := baseValues()
	v.Cluster = ""
	if _, err := CompileForkDeployment(v, NoContext()); err == nil {
		t.Error("expected an error when cluster is missing")
	}

	v = baseValues()
	v.Forks = nil
	if _, err := CompileForkDeployment(v, NoContext()); err == nil {
		t.Error("expected an error when there are no forks")
	}
}

func TestCompileNamespaceResolution(t *testing.T) {
	// Per-fork namespace beats the default.
	v := baseValues()
	v.Forks[0].Namespace = "web"
	sb, err := CompileForkDeployment(v, NoContext())
	if err != nil {
		t.Fatal(err)
	}
	if got := str(sb.Spec.Forks[0].ForkOf.Namespace); got != "web" {
		t.Errorf("got namespace %q", got)
	}

	// With neither, the error names the fork and both ways to fix it.
	v = baseValues()
	v.Defaults = nil
	_, err = CompileForkDeployment(v, NoContext())
	if err == nil || !strings.Contains(err.Error(), `fork "route"`) {
		t.Errorf("got %v", err)
	}
}

// Each fork's forkOf must point at its own workload: the pointers the API models
// use make aliasing a live hazard here.
func TestCompileForkOfPointersAreDistinct(t *testing.T) {
	v := &Values{
		Cluster:  "c",
		Defaults: &ValuesDefaults{Namespace: "hotrod"},
		Forks: []ValuesFork{
			{Workload: "route", Image: "a:1"},
			{Workload: "frontend", Namespace: "web", Image: "b:1", Kind: "Rollout"},
		},
	}
	sb, err := CompileForkDeployment(v, NoContext())
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []struct{ name, ns, kind string }{
		{"route", "hotrod", "Deployment"},
		{"frontend", "web", "Rollout"},
	} {
		got := sb.Spec.Forks[i].ForkOf
		if str(got.Name) != want.name || str(got.Namespace) != want.ns || str(got.Kind) != want.kind {
			t.Errorf("forks[%d]: got %s/%s %s, want %s/%s %s", i,
				str(got.Namespace), str(got.Name), str(got.Kind), want.ns, want.name, want.kind)
		}
	}
}

func TestImagePrecedence(t *testing.T) {
	ctx := DetectGitHub(MapEnv(map[string]string{
		"GITHUB_REPOSITORY": "acme/route",
		"GITHUB_SHA":        "abc1234def5678",
	}))

	t.Run("images beats image and template", func(t *testing.T) {
		v := baseValues()
		v.Defaults.ImageTemplate = "tpl/{workload}:{short-sha}"
		v.Forks[0].Images = []ValuesImage{{Image: "explicit:1", Container: "app"}}
		sb, err := CompileForkDeployment(v, ctx)
		if err != nil {
			t.Fatal(err)
		}
		imgs := sb.Spec.Forks[0].Customizations.Images
		if len(imgs) != 1 || imgs[0].Image != "explicit:1" || imgs[0].Container != "app" {
			t.Errorf("got %+v", imgs)
		}
	})

	t.Run("image beats template", func(t *testing.T) {
		v := baseValues()
		v.Defaults.ImageTemplate = "tpl/{workload}:{short-sha}"
		sb, err := CompileForkDeployment(v, ctx)
		if err != nil {
			t.Fatal(err)
		}
		if sb.Spec.Forks[0].Customizations.Images[0].Image != "acme/route:1" {
			t.Errorf("got %+v", sb.Spec.Forks[0].Customizations.Images)
		}
	})

	t.Run("template fills in", func(t *testing.T) {
		v := baseValues()
		v.Forks[0].Image = ""
		v.Defaults.ImageTemplate = "tpl/{workload}:{short-sha}"
		sb, err := CompileForkDeployment(v, ctx)
		if err != nil {
			t.Fatal(err)
		}
		if got := sb.Spec.Forks[0].Customizations.Images[0].Image; got != "tpl/route:abc1234" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("an env-only fork needs no image", func(t *testing.T) {
		v := baseValues()
		v.Forks[0].Image = ""
		v.Forks[0].Env = map[string]EnvValue{"A": StringEnv("1")}
		sb, err := CompileForkDeployment(v, NoContext())
		if err != nil {
			t.Fatal(err)
		}
		if sb.Spec.Forks[0].Customizations.Images != nil {
			t.Errorf("got %+v", sb.Spec.Forks[0].Customizations.Images)
		}
	})
}

func TestResolveImageTemplate(t *testing.T) {
	ph := map[string]string{"workload": "route", "short-sha": "abc1234", "pr": ""}

	got, err := ResolveImageTemplate("ghcr.io/acme/{workload}:{short-sha}", ph)
	if err != nil {
		t.Fatal(err)
	}
	if got != "ghcr.io/acme/route:abc1234" {
		t.Errorf("got %q", got)
	}

	// A placeholder with no value names itself in the error rather than
	// producing an image reference that would fail an obscure pull.
	_, err = ResolveImageTemplate("ghcr.io/acme/{workload}:{pr}-{branch-slug}", ph)
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"pr", "branch-slug"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should name %q: %v", want, err)
		}
	}
}

func TestMergeEnvAndResources(t *testing.T) {
	v := &Values{
		Cluster: "prod-eks",
		Defaults: &ValuesDefaults{
			Namespace: "hotrod",
			Env:       map[string]EnvValue{"LOG_LEVEL": StringEnv("info"), "SHARED": StringEnv("yes")},
		},
		Resources: []ValuesResource{{Name: "db", Plugin: "mariadb"}},
		Forks: []ValuesFork{{
			Workload: "route",
			Env: map[string]EnvValue{
				"LOG_LEVEL": StringEnv("debug"),
				"HOST":      {FromResource: "db.host", IsRef: true},
				"PORT":      StringEnv("${resource:db.port}"),
				"LITERAL":   StringEnv("$${resource:db.port}"),
				"EMPTY":     StringEnv(""),
			},
		}},
	}
	sb, err := CompileForkDeployment(v, NoContext())
	if err != nil {
		t.Fatal(err)
	}
	env := sb.Spec.Forks[0].Customizations.Env

	// Sorted by name, so the rendered spec is stable across runs.
	var names []string
	for _, e := range env {
		names = append(names, e.Name)
	}
	want := []string{"EMPTY", "HOST", "LITERAL", "LOG_LEVEL", "PORT", "SHARED"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("got %v, want %v", names, want)
	}

	byName := map[string]*models.SandboxEnvVar{}
	for _, e := range env {
		byName[e.Name] = e
	}
	if byName["LOG_LEVEL"].Value != "debug" {
		t.Errorf("per-fork env should win: %+v", byName["LOG_LEVEL"])
	}
	if r := byName["HOST"].ValueFrom; r == nil || r.Resource.Name != "db" || r.Resource.OutputKey != "host" {
		t.Errorf("got %+v", byName["HOST"])
	}
	if r := byName["PORT"].ValueFrom; r == nil || r.Resource.OutputKey != "port" {
		t.Errorf("the ${resource:...} sigil should resolve: %+v", byName["PORT"])
	}
	if byName["LITERAL"].Value != "${resource:db.port}" {
		t.Errorf("$${...} should unescape to a literal: %+v", byName["LITERAL"])
	}
	if byName["EMPTY"].Value != "" || byName["EMPTY"].ValueFrom != nil {
		t.Errorf("an empty value should stay empty: %+v", byName["EMPTY"])
	}
}

func TestUndeclaredResourceReference(t *testing.T) {
	v := baseValues()
	v.Forks[0].Env = map[string]EnvValue{"HOST": {FromResource: "nope.host", IsRef: true}}
	_, err := CompileForkDeployment(v, NoContext())
	if err == nil || !strings.Contains(err.Error(), "undeclared resource") {
		t.Errorf("got %v", err)
	}

	v = baseValues()
	v.Resources = []ValuesResource{{Name: "db", Plugin: "mariadb"}}
	v.Forks[0].Env = map[string]EnvValue{"HOST": {FromResource: "db", IsRef: true}}
	_, err = CompileForkDeployment(v, NoContext())
	if err == nil || !strings.Contains(err.Error(), "name.key") {
		t.Errorf("got %v", err)
	}
}

func TestResourceValidation(t *testing.T) {
	v := baseValues()
	v.Resources = []ValuesResource{{Plugin: "mariadb"}}
	if _, err := CompileForkDeployment(v, NoContext()); err == nil {
		t.Error("expected an error for a resource without a name")
	}

	v = baseValues()
	v.Resources = []ValuesResource{{Name: "db"}}
	if _, err := CompileForkDeployment(v, NoContext()); err == nil {
		t.Error("expected an error for a resource without a plugin")
	}
}

func TestParseValuesRejectsUnknownFields(t *testing.T) {
	// A typo like `envs:` would otherwise be silently dropped, producing a
	// sandbox missing configuration the user asked for.
	_, err := ParseValues([]byte("cluster: c\nforks:\n- workload: route\n  envs:\n    A: 1\n"))
	if err == nil || !strings.Contains(err.Error(), "envs") {
		t.Errorf("got %v", err)
	}

	_, err = ParseValues([]byte("clustr: c\n"))
	if err == nil || !strings.Contains(err.Error(), "clustr") {
		t.Errorf("got %v", err)
	}

	// The message says what is wrong without the decoder's wrapping.
	if got := err.Error(); got != `couldn't parse the values document: unknown field "clustr"` {
		t.Errorf("got %q", got)
	}
}

func TestMergePatch(t *testing.T) {
	dst := map[string]any{
		"name": "sb",
		"spec": map[string]any{
			"cluster": "a",
			"ttl":     map[string]any{"duration": "2d"},
			"forks":   []any{"one"},
		},
	}
	patch := map[string]any{
		"spec": map[string]any{
			"description": "added",
			"ttl":         map[string]any{"duration": "1h"},
			"forks":       []any{"replaced"},
			"cluster":     nil,
		},
	}
	got := MergePatch(dst, patch)
	spec := got["spec"].(map[string]any)
	if _, ok := spec["cluster"]; ok {
		t.Error("an explicit null should delete the key")
	}
	if spec["description"] != "added" {
		t.Errorf("got %+v", spec)
	}
	if spec["ttl"].(map[string]any)["duration"] != "1h" {
		t.Error("maps should merge recursively")
	}
	if len(spec["forks"].([]any)) != 1 || spec["forks"].([]any)[0] != "replaced" {
		t.Error("lists should replace wholesale")
	}
}

// Several fields in the API models lack omitempty, so a compiled sandbox
// marshals with nulls that have to be pruned before the document is shown or
// patched.
func TestSandboxToDocPrunesNulls(t *testing.T) {
	sb, err := CompileForkDeployment(baseValues(), NoContext())
	if err != nil {
		t.Fatal(err)
	}
	sb.Name = "sb"
	doc, err := SandboxToDoc(sb)
	if err != nil {
		t.Fatal(err)
	}
	spec := doc["spec"].(map[string]any)
	for _, k := range []string{"local", "virtual", "endpoints", "middleware", "resources", "ttl"} {
		if _, ok := spec[k]; ok {
			t.Errorf("expected %q to be pruned, got %+v", k, spec[k])
		}
	}
	if spec["cluster"] != "prod-eks" {
		t.Errorf("got %+v", spec)
	}
}
