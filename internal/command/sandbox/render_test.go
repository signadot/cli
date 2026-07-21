package sandbox

import (
	"strings"
	"testing"

	"github.com/signadot/cli/internal/config"
)

func noneCtx() *CIContext { return &CIContext{Provider: contextNone} }

func TestCompileSingleFork(t *testing.T) {
	vals := &Values{
		Cluster:  "prod-eks",
		Defaults: &ValuesDefaults{Namespace: "hotrod"},
		Forks: []ValuesFork{
			{Workload: "route", Image: "ghcr.io/acme/route:abc123"},
		},
	}
	sb, err := compileForkDeployment(vals, noneCtx())
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if got := *sb.Spec.Cluster; got != "prod-eks" {
		t.Errorf("cluster = %q", got)
	}
	if len(sb.Spec.Forks) != 1 {
		t.Fatalf("forks = %d", len(sb.Spec.Forks))
	}
	f := sb.Spec.Forks[0]
	if *f.ForkOf.Name != "route" || *f.ForkOf.Kind != "Deployment" || *f.ForkOf.Namespace != "hotrod" {
		t.Errorf("forkOf = %+v", f.ForkOf)
	}
	if len(f.Customizations.Images) != 1 || f.Customizations.Images[0].Image != "ghcr.io/acme/route:abc123" {
		t.Errorf("images = %+v", f.Customizations.Images)
	}
}

func TestCompileImageTemplateAndFromResource(t *testing.T) {
	vals := &Values{
		Cluster: "prod-eks",
		Defaults: &ValuesDefaults{
			Namespace:     "hotrod",
			ImageTemplate: "ghcr.io/acme/{workload}:{sha}",
		},
		Resources: []ValuesResource{
			{Name: "customerdb", Plugin: "hotrod-mariadb", Params: map[string]string{"dbname": "customer"}},
		},
		Forks: []ValuesFork{
			{
				Workload: "customer",
				Env: map[string]EnvValue{
					"DB_HOST": {IsFrom: true, FromResource: "customerdb.host"},
				},
			},
			{Workload: "route"},
		},
	}
	cctx := &CIContext{Provider: contextGitHub, Detected: true, SHA: "abc123"}
	sb, err := compileForkDeployment(vals, cctx)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if len(sb.Spec.Resources) != 1 || sb.Spec.Resources[0].Plugin != "hotrod-mariadb" {
		t.Errorf("resources = %+v", sb.Spec.Resources)
	}
	customer := sb.Spec.Forks[0]
	if img := customer.Customizations.Images[0].Image; img != "ghcr.io/acme/customer:abc123" {
		t.Errorf("templated image = %q", img)
	}
	ev := customer.Customizations.Env[0]
	if ev.Name != "DB_HOST" || ev.ValueFrom == nil || ev.ValueFrom.Resource == nil {
		t.Fatalf("env = %+v", ev)
	}
	if ev.ValueFrom.Resource.Name != "customerdb" || ev.ValueFrom.Resource.OutputKey != "host" {
		t.Errorf("resource ref = %+v", ev.ValueFrom.Resource)
	}
	// route fork resolves its own workload placeholder
	if img := sb.Spec.Forks[1].Customizations.Images[0].Image; img != "ghcr.io/acme/route:abc123" {
		t.Errorf("route image = %q", img)
	}
}

func TestUnresolvableImagePlaceholder(t *testing.T) {
	vals := &Values{
		Cluster:  "c",
		Defaults: &ValuesDefaults{Namespace: "ns", ImageTemplate: "acme/{workload}:{sha}"},
		Forks:    []ValuesFork{{Workload: "route"}},
	}
	_, err := compileForkDeployment(vals, noneCtx())
	if err == nil || !strings.Contains(err.Error(), "sha") {
		t.Fatalf("expected unresolvable {sha} error, got %v", err)
	}
}

func TestUndeclaredFromResource(t *testing.T) {
	vals := &Values{
		Cluster:  "c",
		Defaults: &ValuesDefaults{Namespace: "ns"},
		Forks: []ValuesFork{{
			Workload: "route",
			Env:      map[string]EnvValue{"X": {IsFrom: true, FromResource: "missing.host"}},
		}},
	}
	_, err := compileForkDeployment(vals, noneCtx())
	if err == nil || !strings.Contains(err.Error(), "undeclared") {
		t.Fatalf("expected undeclared resource error, got %v", err)
	}
}

func TestEnvSigilAndEscape(t *testing.T) {
	vals := &Values{
		Cluster:   "c",
		Defaults:  &ValuesDefaults{Namespace: "ns"},
		Resources: []ValuesResource{{Name: "db", Plugin: "p"}},
		Forks: []ValuesFork{{
			Workload: "route",
			Env: map[string]EnvValue{
				"A_REF":     {String: "${resource:db.host}"},
				"B_LITERAL": {String: "$${resource:db.host}"},
				"C_PLAIN":   {String: "debug"},
			},
		}},
	}
	sb, err := compileForkDeployment(vals, noneCtx())
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	env := map[string]struct {
		val  string
		from bool
	}{}
	for _, e := range sb.Spec.Forks[0].Customizations.Env {
		if e.ValueFrom != nil && e.ValueFrom.Resource != nil {
			env[e.Name] = struct {
				val  string
				from bool
			}{e.ValueFrom.Resource.Name + "." + e.ValueFrom.Resource.OutputKey, true}
		} else {
			env[e.Name] = struct {
				val  string
				from bool
			}{e.Value, false}
		}
	}
	if e := env["A_REF"]; !e.from || e.val != "db.host" {
		t.Errorf("A_REF = %+v", e)
	}
	if e := env["B_LITERAL"]; e.from || e.val != "${resource:db.host}" {
		t.Errorf("B_LITERAL = %+v", e)
	}
	if e := env["C_PLAIN"]; e.from || e.val != "debug" {
		t.Errorf("C_PLAIN = %+v", e)
	}
}

func TestEnvMergeForkWins(t *testing.T) {
	vals := &Values{
		Cluster: "c",
		Defaults: &ValuesDefaults{
			Namespace: "ns",
			Env: map[string]EnvValue{
				"DEPLOY_ENV": {String: "sandbox"},
				"LOG_LEVEL":  {String: "info"},
			},
		},
		Forks: []ValuesFork{{
			Workload: "route",
			Env:      map[string]EnvValue{"LOG_LEVEL": {String: "debug"}},
		}},
	}
	sb, err := compileForkDeployment(vals, noneCtx())
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	got := map[string]string{}
	for _, e := range sb.Spec.Forks[0].Customizations.Env {
		got[e.Name] = e.Value
	}
	if got["DEPLOY_ENV"] != "sandbox" || got["LOG_LEVEL"] != "debug" {
		t.Errorf("merged env = %+v", got)
	}
}

func TestParseValuesEnvForms(t *testing.T) {
	doc := []byte(`
cluster: prod-eks
defaults:
  namespace: hotrod
resources:
  - name: customerdb
    plugin: hotrod-mariadb
forks:
  - workload: route
    env:
      LOG_LEVEL: debug
      DB_HOST: { fromResource: customerdb.host }
`)
	vals, err := parseValues(doc)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	env := vals.Forks[0].Env
	if env["LOG_LEVEL"].String != "debug" || env["LOG_LEVEL"].IsFrom {
		t.Errorf("LOG_LEVEL = %+v", env["LOG_LEVEL"])
	}
	if !env["DB_HOST"].IsFrom || env["DB_HOST"].FromResource != "customerdb.host" {
		t.Errorf("DB_HOST = %+v", env["DB_HOST"])
	}
}

func TestNormalizeName(t *testing.T) {
	short := normalizeName("Acme/Route")
	if short != "acme-route" {
		t.Errorf("slug = %q", short)
	}
	long := normalizeName(strings.Repeat("verylongreponame", 4) + "-pr-1234")
	if len(long) > maxNameLen {
		t.Errorf("normalized name %q exceeds %d bytes", long, maxNameLen)
	}
	if !strings.Contains(long, "-") {
		t.Errorf("expected hash-suffixed name, got %q", long)
	}
}

func TestApplyContextGitHub(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITHUB_REPOSITORY", "acme/route")
	t.Setenv("GITHUB_REF", "refs/pull/12/merge")
	t.Setenv("GITHUB_SHA", "abcdef1234567890")
	t.Setenv("GITHUB_HEAD_REF", "feature/new-thing")

	cctx, err := detectCIContext(contextAuto)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if cctx.Provider != contextGitHub || cctx.PR != "12" || cctx.RepoSlug != "route" {
		t.Fatalf("ctx = %+v", cctx)
	}

	vals := &Values{Cluster: "c", Defaults: &ValuesDefaults{Namespace: "ns"}, Forks: []ValuesFork{{Workload: "route", Image: "i:t"}}}
	sb, err := compileForkDeployment(vals, cctx)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if err := applyContext(sb, cctx, "", ""); err != nil {
		t.Fatalf("applyContext: %v", err)
	}
	if sb.Name != "route-pr-12" {
		t.Errorf("name = %q", sb.Name)
	}
	if sb.Spec.Labels[usageLabelKey] != "ci" {
		t.Errorf("usage label = %q", sb.Spec.Labels[usageLabelKey])
	}
	if sb.Spec.Labels["signadot/github-repo"] != "acme/route" || sb.Spec.Labels["signadot/github-pull-request"] != "12" {
		t.Errorf("provider labels = %+v", sb.Spec.Labels)
	}
	if sb.Spec.TTL == nil || sb.Spec.TTL.Duration != defaultCITTL {
		t.Errorf("ttl = %+v", sb.Spec.TTL)
	}
}

func TestApplyContextNoneRequiresName(t *testing.T) {
	sb, err := compileForkDeployment(&Values{
		Cluster:  "c",
		Defaults: &ValuesDefaults{Namespace: "ns"},
		Forks:    []ValuesFork{{Workload: "route", Image: "i:t"}},
	}, noneCtx())
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if err := applyContext(sb, noneCtx(), "", ""); err == nil {
		t.Fatalf("expected error when no name and no CI context")
	}
	if err := applyContext(sb, noneCtx(), "my-sbx", ""); err != nil {
		t.Fatalf("applyContext with explicit name: %v", err)
	}
	if sb.Name != "my-sbx" {
		t.Errorf("name = %q", sb.Name)
	}
}

func TestParseForkSpec(t *testing.T) {
	f, err := parseForkSpec("route")
	if err != nil || f.Workload != "route" {
		t.Fatalf("bare: %+v err=%v", f, err)
	}

	f, err = parseForkSpec("route,image=ghcr.io/acme/route:{sha},namespace=hotrod,kind=Rollout")
	if err != nil {
		t.Fatalf("attrs: %v", err)
	}
	if f.Workload != "route" || f.Image != "ghcr.io/acme/route:{sha}" || f.Namespace != "hotrod" || f.Kind != "Rollout" {
		t.Errorf("attrs = %+v", f)
	}

	f, err = parseForkSpec("workload=route,image=i:t")
	if err != nil || f.Workload != "route" || f.Image != "i:t" {
		t.Fatalf("workload= form: %+v err=%v", f, err)
	}

	if _, err := parseForkSpec("route,bogus=x"); err == nil || !strings.Contains(err.Error(), "unknown attribute") {
		t.Fatalf("expected unknown attribute error, got %v", err)
	}
	if _, err := parseForkSpec("image=i:t"); err == nil || !strings.Contains(err.Error(), "workload name is required") {
		t.Fatalf("expected missing workload error, got %v", err)
	}
	if _, err := parseForkSpec("route,frontend"); err == nil || !strings.Contains(err.Error(), "unexpected token") {
		t.Fatalf("expected unexpected-token error, got %v", err)
	}
}

func TestBuildValuesMultiForkImageTemplate(t *testing.T) {
	cfg := &config.SandboxRender{
		Cluster:       "prod-eks",
		Namespace:     "hotrod",
		Kind:          "Deployment",
		ImageTemplate: "ghcr.io/acme/{workload}:{sha}",
		Forks:         []string{"route", "frontend,namespace=web"},
	}
	vals, err := buildValues(cfg)
	if err != nil {
		t.Fatalf("buildValues: %v", err)
	}
	if vals.Cluster != "prod-eks" {
		t.Errorf("cluster = %q", vals.Cluster)
	}
	if vals.Defaults == nil || vals.Defaults.Namespace != "hotrod" || vals.Defaults.ImageTemplate != "ghcr.io/acme/{workload}:{sha}" {
		t.Fatalf("defaults = %+v", vals.Defaults)
	}
	if len(vals.Forks) != 2 {
		t.Fatalf("forks = %d", len(vals.Forks))
	}
	if vals.Forks[1].Workload != "frontend" || vals.Forks[1].Namespace != "web" {
		t.Errorf("fork[1] = %+v", vals.Forks[1])
	}

	cctx := &CIContext{Provider: contextGitHub, Detected: true, SHA: "abc123"}
	sb, err := compileForkDeployment(vals, cctx)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if img := sb.Spec.Forks[0].Customizations.Images[0].Image; img != "ghcr.io/acme/route:abc123" {
		t.Errorf("route image = %q", img)
	}
	if img := sb.Spec.Forks[1].Customizations.Images[0].Image; img != "ghcr.io/acme/frontend:abc123" {
		t.Errorf("frontend image = %q", img)
	}
	if ns := *sb.Spec.Forks[1].ForkOf.Namespace; ns != "web" {
		t.Errorf("frontend ns = %q", ns)
	}
}

func TestBuildValuesSingleForkImage(t *testing.T) {
	cfg := &config.SandboxRender{
		Cluster:   "c",
		Namespace: "ns",
		Kind:      "Deployment",
		Image:     "img:tag",
		Forks:     []string{"route"},
	}
	vals, err := buildValues(cfg)
	if err != nil {
		t.Fatalf("buildValues: %v", err)
	}
	if vals.Forks[0].Image != "img:tag" {
		t.Errorf("image = %q", vals.Forks[0].Image)
	}
}

func TestBuildValuesImageWithMultipleForks(t *testing.T) {
	cfg := &config.SandboxRender{
		Cluster: "c",
		Image:   "img:tag",
		Forks:   []string{"route", "frontend"},
	}
	if _, err := buildValues(cfg); err == nil || !strings.Contains(err.Error(), "single --fork") {
		t.Fatalf("expected --image single-fork error, got %v", err)
	}
}

func TestBuildValuesImageConflict(t *testing.T) {
	cfg := &config.SandboxRender{
		Cluster: "c",
		Image:   "img:tag",
		Forks:   []string{"route,image=other:tag"},
	}
	if _, err := buildValues(cfg); err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("expected image conflict error, got %v", err)
	}
}

func TestBuildValuesNothingToRender(t *testing.T) {
	if _, err := buildValues(&config.SandboxRender{Cluster: "c"}); err == nil || !strings.Contains(err.Error(), "nothing to render") {
		t.Fatalf("expected nothing-to-render error, got %v", err)
	}
}

func TestMergePatch(t *testing.T) {
	dst := map[string]any{
		"name": "route-pr-12",
		"spec": map[string]any{
			"cluster": "prod-eks",
			"ttl":     map[string]any{"duration": "2d"},
		},
	}
	patch := map[string]any{
		"spec": map[string]any{
			"description": "from patch",
			"ttl":         map[string]any{"duration": "1h"},
		},
	}
	out := mergePatch(dst, patch)
	spec := out["spec"].(map[string]any)
	if spec["description"] != "from patch" {
		t.Errorf("description not merged: %+v", spec)
	}
	if spec["cluster"] != "prod-eks" {
		t.Errorf("cluster clobbered: %+v", spec)
	}
	if spec["ttl"].(map[string]any)["duration"] != "1h" {
		t.Errorf("ttl not deep-merged: %+v", spec["ttl"])
	}
}
