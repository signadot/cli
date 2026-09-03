package render

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
)

// BuiltinTemplateName identifies the built-in template, so that a rendered spec
// can be traced back to the rules that produced it and so the shape can be
// revised later without changing what existing callers get.
const BuiltinTemplateName = "fork-deployment@v1"

//go:embed fork-deployment.yaml
var builtinTemplate []byte

// BuiltinTemplate returns the source of the built-in template, for
// `signadot sandbox template show`.
func BuiltinTemplate() []byte { return builtinTemplate }

// Options controls rendering beyond the values document itself.
type Options struct {
	// Name overrides the name in the values document.
	Name string
	// TTL overrides the TTL in the values document.
	TTL string
	// DefaultLabels controls whether the built-in signadot/* labels are
	// stamped. It only has an effect when a CI context was detected.
	DefaultLabels bool
}

// RenderValues turns a values document into the sandbox document that gets
// applied: it compiles the values, resolves the name, TTL and labels against the
// CI context, and renders the built-in template with the result.
func RenderValues(vals *Values, ctx CIContext, opts Options) (map[string]any, error) {
	sb, err := CompileForkDeployment(vals, ctx)
	if err != nil {
		return nil, err
	}

	// A values document is an input we compile, not a spec we pass through, so a
	// name it carries is normalised just like one from --name. That matters
	// because these names typically come from CI variables. A spec document's
	// name, by contrast, is left exactly as authored.
	name := opts.Name
	if name == "" {
		name, sb.Name = sb.Name, ""
	}
	if err := ApplyContext(sb, ctx, name, opts.TTL, opts.DefaultLabels); err != nil {
		return nil, err
	}
	if err := ValidateSandbox(sb); err != nil {
		return nil, err
	}
	return renderBuiltin(sb)
}

// renderBuiltin puts the compiled sandbox through the template engine rather
// than marshalling it directly, so that the built-in path and a user's own
// template path render identically and the built-in template stays honest: what
// `template show` prints is what produced the output.
func renderBuiltin(sb *models.Sandbox) (map[string]any, error) {
	vars, err := templateVars(sb)
	if err != nil {
		return nil, err
	}
	out, err := utils.RenderTemplate(builtinTemplate, vars, utils.TemplateOptions{})
	if err != nil {
		return nil, fmt.Errorf("couldn't render the %s template: %w", BuiltinTemplateName, err)
	}
	doc, ok := PruneNulls(out).(map[string]any)
	if !ok {
		return nil, fmt.Errorf("the %s template did not render to a mapping", BuiltinTemplateName)
	}
	return doc, nil
}

// templateVars binds one variable per field of the built-in template.
func templateVars(sb *models.Sandbox) (config.TemplateVals, error) {
	spec := sb.Spec
	if spec == nil {
		spec = &models.SandboxSpec{}
	}
	var cluster string
	if spec.Cluster != nil {
		cluster = *spec.Cluster
	}

	fields := []struct {
		name string
		val  any
	}{
		{"name", sb.Name},
		{"cluster", cluster},
		{"description", spec.Description},
		{"labels", spec.Labels},
		{"ttl", spec.TTL},
		{"resources", spec.Resources},
		{"forks", spec.Forks},
		{"defaultRouteGroup", spec.DefaultRouteGroup},
	}

	vals := make(config.TemplateVals, 0, len(fields))
	for _, f := range fields {
		enc, err := encodeVar(f.val)
		if err != nil {
			return nil, fmt.Errorf("couldn't encode %q: %w", f.name, err)
		}
		vals = append(vals, config.TemplateVal{Var: f.name, Val: enc})
	}
	return vals, nil
}

// encodeVar renders a value as the JSON that the template engine's [yaml]
// encoding will parse back into the same structure. An unset value encodes as
// the empty string, which renders as null so the key can be pruned.
func encodeVar(v any) (string, error) {
	if isUnset(v) {
		return "", nil
	}
	d, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(d), nil
}

func isUnset(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String:
		return rv.Len() == 0
	case reflect.Slice, reflect.Map:
		return rv.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return rv.IsNil()
	default:
		return false
	}
}
