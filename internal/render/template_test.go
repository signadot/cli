package render

import (
	"regexp"
	"strings"
	"testing"

	"github.com/signadot/go-sdk/models"
)

var templateVarRx = regexp.MustCompile(`@\{\s*([a-zA-Z][a-zA-Z0-9_.-]*)\s*\[`)

// The template engine fails on a variable it has no value for, so a variable
// added to the built-in template without a matching binding would break every
// values document rather than just the ones using the new field.
func TestBuiltinTemplateVariablesAreAllBound(t *testing.T) {
	var inTemplate []string
	for _, m := range templateVarRx.FindAllStringSubmatch(string(builtinTemplate), -1) {
		inTemplate = append(inTemplate, m[1])
	}
	if len(inTemplate) == 0 {
		t.Fatal("found no variables in the built-in template")
	}

	vars, err := templateVars(&models.Sandbox{})
	if err != nil {
		t.Fatal(err)
	}
	bound := map[string]bool{}
	for _, v := range vars {
		bound[v.Var] = true
	}
	for _, name := range inTemplate {
		if !bound[name] {
			t.Errorf("the built-in template uses @{%s} but templateVars does not bind it", name)
		}
	}

	// And the reverse: a binding with nothing to fill is dead weight.
	for _, v := range vars {
		if !strings.Contains(string(builtinTemplate), "@{"+v.Var+"[") {
			t.Errorf("templateVars binds %q, which the built-in template does not use", v.Var)
		}
	}
}

// Rendering only emits what the values document set, so that a spec is not
// littered with empty fields a reader has to skip over.
func TestRenderValuesOmitsUnsetFields(t *testing.T) {
	doc, err := RenderValues(baseValues(), NoContext(), Options{Name: "sb"})
	if err != nil {
		t.Fatal(err)
	}
	spec, ok := doc["spec"].(map[string]any)
	if !ok {
		t.Fatalf("got %+v", doc)
	}
	for _, k := range []string{"description", "labels", "ttl", "resources", "defaultRouteGroup"} {
		if _, present := spec[k]; present {
			t.Errorf("expected %q to be absent, got %+v", k, spec[k])
		}
	}
	if spec["cluster"] != "prod-eks" || doc["name"] != "sb" {
		t.Errorf("got %+v", doc)
	}
}

// A values document's name is normalised, unlike a name authored in a spec,
// because it usually reaches the values document from a CI variable.
func TestRenderValuesNormalizesItsOwnName(t *testing.T) {
	v := baseValues()
	v.Name = "Feature/Fix_Routing"
	doc, err := RenderValues(v, NoContext(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if doc["name"] != "feature-fix-routing" {
		t.Errorf("got %+v", doc["name"])
	}

	// --name still wins over it.
	doc, err = RenderValues(v, NoContext(), Options{Name: "override"})
	if err != nil {
		t.Fatal(err)
	}
	if doc["name"] != "override" {
		t.Errorf("got %+v", doc["name"])
	}
}
