package sandbox

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/signadot/go-sdk/models"
	sigsyaml "sigs.k8s.io/yaml"
)

// usageLabelKey is the reserved, tooling-set label described in the design (§4).
// It uses the product's reserved "signadot/" label namespace, consistent with
// the App's other reserved keys (e.g. signadot/github-repo).
const usageLabelKey = "signadot/usage"

// Values is the schema v1 document shared across every CI integration (§3).
// It is produced by wrappers from their native inputs and consumed by
// `sandbox render --values`.
type Values struct {
	Name      string            `json:"name,omitempty"`
	Cluster   string            `json:"cluster,omitempty"`
	Defaults  *ValuesDefaults   `json:"defaults,omitempty"`
	TTL       string            `json:"ttl,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	Resources []ValuesResource  `json:"resources,omitempty"`
	Forks     []ValuesFork      `json:"forks,omitempty"`
}

type ValuesDefaults struct {
	Namespace     string              `json:"namespace,omitempty"`
	ImageTemplate string              `json:"imageTemplate,omitempty"`
	Env           map[string]EnvValue `json:"env,omitempty"`
}

type ValuesResource struct {
	Name   string            `json:"name"`
	Plugin string            `json:"plugin"`
	Params map[string]string `json:"params,omitempty"`
}

type ValuesImage struct {
	Container string `json:"container,omitempty"`
	Image     string `json:"image,omitempty"`
}

type ValuesEndpoint struct {
	Name     string `json:"name"`
	Port     int64  `json:"port"`
	Protocol string `json:"protocol,omitempty"`
}

type ValuesFork struct {
	Workload  string                 `json:"workload"`
	Kind      string                 `json:"kind,omitempty"`
	Namespace string                 `json:"namespace,omitempty"`
	Image     string                 `json:"image,omitempty"`
	Images    []ValuesImage          `json:"images,omitempty"`
	Env       map[string]EnvValue    `json:"env,omitempty"`
	Endpoints []ValuesEndpoint       `json:"endpoints,omitempty"`
	Patch     map[string]interface{} `json:"patch,omitempty"`
}

// EnvValue is either a plain string or the structured {fromResource: name.key}
// form. String values may additionally embed the ${resource:name.key} sigil
// used in flat-string contexts (§3).
type EnvValue struct {
	String       string
	FromResource string
	IsFrom       bool
}

func (e *EnvValue) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		e.String = s
		return nil
	}
	var obj struct {
		FromResource string `json:"fromResource"`
	}
	if err := json.Unmarshal(data, &obj); err == nil && obj.FromResource != "" {
		e.FromResource = obj.FromResource
		e.IsFrom = true
		return nil
	}
	return fmt.Errorf("env value must be a string or {fromResource: name.key}")
}

var (
	imagePlaceholderRx = regexp.MustCompile(`\{([a-zA-Z][a-zA-Z0-9-]*)\}`)
	// fullResourceSigilRx matches a value that is exactly ${resource:name.key}.
	fullResourceSigilRx = regexp.MustCompile(`^\$\{resource:([^}]+)\}$`)
)

// parseValues loads a values document (YAML or JSON) into the schema v1 struct.
func parseValues(data []byte) (*Values, error) {
	vals := &Values{}
	if err := sigsyaml.UnmarshalStrict(data, vals); err != nil {
		return nil, fmt.Errorf("couldn't parse values document: %s",
			strings.TrimPrefix(err.Error(), "error unmarshaling JSON: while decoding JSON: "))
	}
	return vals, nil
}

// parseForkSpec parses a single self-contained fork record into a ValuesFork.
// The first bare (no '=') comma-separated token is the workload name; remaining
// key=value tokens set per-fork attributes. Commas are attribute separators, so
// each fork is specified on its own (one --fork flag / one input line).
//
// Examples:
//
//	route
//	route,image=ghcr.io/acme/route:{sha}
//	workload=route,namespace=hotrod,kind=Deployment,image=ghcr.io/acme/route:{sha}
func parseForkSpec(spec string) (ValuesFork, error) {
	var f ValuesFork
	for i, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, val, hasEq := strings.Cut(part, "=")
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if !hasEq {
			if i != 0 {
				return f, fmt.Errorf("invalid fork spec %q: unexpected token %q; specify one fork at a time and use key=value for attributes (image, namespace, kind)", spec, part)
			}
			f.Workload = part
			continue
		}
		switch key {
		case "workload":
			f.Workload = val
		case "image":
			f.Image = val
		case "namespace":
			f.Namespace = val
		case "kind":
			f.Kind = val
		default:
			return f, fmt.Errorf("invalid fork spec %q: unknown attribute %q (allowed: workload, image, namespace, kind)", spec, key)
		}
	}
	if f.Workload == "" {
		return f, fmt.Errorf("invalid fork spec %q: workload name is required", spec)
	}
	return f, nil
}

// compileForkDeployment renders a Values document into a Sandbox using the
// built-in fork-deployment@v1 template rules (§3).
func compileForkDeployment(vals *Values, cctx *CIContext) (*models.Sandbox, error) {
	if vals.Cluster == "" {
		return nil, fmt.Errorf("cluster is required (set --cluster or values.cluster)")
	}
	if len(vals.Forks) == 0 {
		return nil, fmt.Errorf("at least one fork is required")
	}

	declared := map[string]bool{}
	var resources []*models.SandboxResource
	for i := range vals.Resources {
		r := &vals.Resources[i]
		if r.Name == "" {
			return nil, fmt.Errorf("resources[%d]: name is required", i)
		}
		if r.Plugin == "" {
			return nil, fmt.Errorf("resource %q: plugin is required", r.Name)
		}
		declared[r.Name] = true
		resources = append(resources, &models.SandboxResource{
			Name:   r.Name,
			Plugin: r.Plugin,
			Params: r.Params,
		})
	}

	var (
		defNS  string
		imgTpl string
		defEnv map[string]EnvValue
	)
	if vals.Defaults != nil {
		defNS = vals.Defaults.Namespace
		imgTpl = vals.Defaults.ImageTemplate
		defEnv = vals.Defaults.Env
	}

	var forks []*models.SandboxFork
	for i := range vals.Forks {
		f := &vals.Forks[i]
		if f.Workload == "" {
			return nil, fmt.Errorf("forks[%d]: workload is required", i)
		}
		ns := f.Namespace
		if ns == "" {
			ns = defNS
		}
		if ns == "" {
			return nil, fmt.Errorf("fork %q: namespace is required (set fork.namespace or defaults.namespace)", f.Workload)
		}
		kind := f.Kind
		if kind == "" {
			kind = "Deployment"
		}

		cust := &models.SandboxCustomizations{}

		images, err := resolveImages(f, imgTpl, cctx, ns)
		if err != nil {
			return nil, err
		}
		cust.Images = images

		env, err := mergeEnv(defEnv, f.Env, declared)
		if err != nil {
			return nil, fmt.Errorf("fork %q: %w", f.Workload, err)
		}
		cust.Env = env

		if len(f.Patch) > 0 {
			patchYAML, err := sigsyaml.Marshal(f.Patch)
			if err != nil {
				return nil, fmt.Errorf("fork %q: couldn't encode patch: %w", f.Workload, err)
			}
			cust.Patch = &models.SandboxCustomPatch{
				Type:  models.SandboxesPatchTypeStrategic,
				Value: string(patchYAML),
			}
		}

		fork := &models.SandboxFork{
			ForkOf: &models.SandboxForkOf{
				Kind:      strPtr(kind),
				Name:      strPtr(f.Workload),
				Namespace: strPtr(ns),
			},
			Customizations: cust,
		}
		for j := range f.Endpoints {
			ep := &f.Endpoints[j]
			if ep.Name == "" {
				return nil, fmt.Errorf("fork %q: endpoints[%d]: name is required", f.Workload, j)
			}
			fork.Endpoints = append(fork.Endpoints, &models.SandboxForkEndpoint{
				Name:     ep.Name,
				Port:     ep.Port,
				Protocol: ep.Protocol,
			})
		}
		forks = append(forks, fork)
	}

	sb := &models.Sandbox{
		Name: vals.Name,
		Spec: &models.SandboxSpec{
			Cluster:   strPtr(vals.Cluster),
			Forks:     forks,
			Resources: resources,
		},
	}
	if len(vals.Labels) > 0 {
		sb.Spec.Labels = map[string]string{}
		for k, v := range vals.Labels {
			sb.Spec.Labels[k] = v
		}
	}
	if vals.TTL != "" {
		sb.Spec.TTL = &models.SandboxTTL{Duration: vals.TTL}
	}
	return sb, nil
}

// resolveImages implements the per-fork image resolution precedence:
// images > image > defaults.imageTemplate > none (§3).
func resolveImages(f *ValuesFork, imgTpl string, cctx *CIContext, ns string) ([]*models.SandboxImage, error) {
	if len(f.Images) > 0 {
		out := make([]*models.SandboxImage, 0, len(f.Images))
		for i := range f.Images {
			img := &f.Images[i]
			if img.Image == "" {
				return nil, fmt.Errorf("fork %q: images[%d]: image is required", f.Workload, i)
			}
			out = append(out, &models.SandboxImage{Container: img.Container, Image: img.Image})
		}
		return out, nil
	}
	if f.Image != "" {
		return []*models.SandboxImage{{Image: f.Image}}, nil
	}
	if imgTpl != "" {
		resolved, err := resolveImageTemplate(imgTpl, cctx.imagePlaceholders(f.Workload, ns))
		if err != nil {
			return nil, fmt.Errorf("fork %q: %w", f.Workload, err)
		}
		return []*models.SandboxImage{{Image: resolved}}, nil
	}
	// env-only fork (legal, useful)
	return nil, nil
}

// resolveImageTemplate substitutes {placeholder} tokens, failing (naming the
// placeholder) when a referenced value is unavailable.
func resolveImageTemplate(tpl string, ph map[string]string) (string, error) {
	var missing []string
	out := imagePlaceholderRx.ReplaceAllStringFunc(tpl, func(tok string) string {
		key := tok[1 : len(tok)-1]
		v, ok := ph[key]
		if !ok || v == "" {
			missing = append(missing, key)
			return tok
		}
		return v
	})
	if len(missing) > 0 {
		return "", fmt.Errorf("unresolvable image-template placeholder(s): %s", strings.Join(dedupe(missing), ", "))
	}
	return out, nil
}

// mergeEnv merges defaults.env under per-fork env (per-fork wins) and compiles
// each value to a SandboxEnvVar, resolving fromResource / ${resource:...} refs.
func mergeEnv(defEnv, forkEnv map[string]EnvValue, declared map[string]bool) ([]*models.SandboxEnvVar, error) {
	if len(defEnv) == 0 && len(forkEnv) == 0 {
		return nil, nil
	}
	merged := map[string]EnvValue{}
	for k, v := range defEnv {
		merged[k] = v
	}
	for k, v := range forkEnv {
		merged[k] = v
	}
	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]*models.SandboxEnvVar, 0, len(keys))
	for _, k := range keys {
		v := merged[k]
		ev := &models.SandboxEnvVar{Name: k}

		switch {
		case v.IsFrom:
			ref, err := resourceRef(v.FromResource, declared)
			if err != nil {
				return nil, fmt.Errorf("env %q: %w", k, err)
			}
			ev.ValueFrom = &models.SandboxEnvValueFrom{Resource: ref}
		default:
			if m := fullResourceSigilRx.FindStringSubmatch(v.String); m != nil {
				ref, err := resourceRef(m[1], declared)
				if err != nil {
					return nil, fmt.Errorf("env %q: %w", k, err)
				}
				ev.ValueFrom = &models.SandboxEnvValueFrom{Resource: ref}
			} else {
				// unescape $${...} -> ${...}
				ev.Value = strings.ReplaceAll(v.String, "$${", "${")
			}
		}
		out = append(out, ev)
	}
	return out, nil
}

// resourceRef parses a "name.key" reference and validates the resource is
// declared in the values document.
func resourceRef(ref string, declared map[string]bool) (*models.SandboxEnvValueFromResource, error) {
	name, key, found := strings.Cut(ref, ".")
	if !found || name == "" || key == "" {
		return nil, fmt.Errorf("resource reference %q must be of the form name.key", ref)
	}
	if !declared[name] {
		return nil, fmt.Errorf("resource reference %q names an undeclared resource %q", ref, name)
	}
	return &models.SandboxEnvValueFromResource{Name: name, OutputKey: key}, nil
}

func strPtr(s string) *string { return &s }

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
