package render

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/signadot/go-sdk/models"
	"sigs.k8s.io/yaml"
)

var imagePlaceholderRx = regexp.MustCompile(`\{([a-zA-Z][a-zA-Z0-9-]*)\}`)

// fullResourceSigilRx matches a value that is exactly ${resource:name.key}.
var fullResourceSigilRx = regexp.MustCompile(`^\$\{resource:([^}]+)\}$`)

// CompileForkDeployment turns a values document into a sandbox, applying the
// defaulting and precedence rules of the built-in fork-deployment template.
//
// This is the part of rendering the template engine cannot do: it has to build a
// list whose length comes from the input, resolve image templates, and merge
// per-fork settings over defaults. The engine substitutes the result.
func CompileForkDeployment(vals *Values, ctx CIContext) (*models.Sandbox, error) {
	if vals.Cluster == "" {
		return nil, fmt.Errorf("cluster is required (set values.cluster)")
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
		out := &models.SandboxResource{Name: r.Name, Plugin: r.Plugin}
		if len(r.Params) > 0 {
			out.Params = copyMap(r.Params)
		}
		resources = append(resources, out)
	}

	var defNS, imgTpl string
	var defEnv map[string]EnvValue
	if vals.Defaults != nil {
		defNS = vals.Defaults.Namespace
		imgTpl = vals.Defaults.ImageTemplate
		defEnv = vals.Defaults.Env
	}

	forks := make([]*models.SandboxFork, 0, len(vals.Forks))
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
			return nil, fmt.Errorf("fork %q: namespace is required "+
				"(set the fork's namespace or defaults.namespace)", f.Workload)
		}
		kind := f.Kind
		if kind == "" {
			kind = "Deployment"
		}

		customizations := &models.SandboxCustomizations{}
		images, err := resolveImages(f, imgTpl, ctx, ns)
		if err != nil {
			return nil, err
		}
		customizations.Images = images

		env, err := mergeEnv(defEnv, f.Env, declared, f.Workload)
		if err != nil {
			return nil, err
		}
		customizations.Env = env

		if len(f.Patch) > 0 {
			d, err := yaml.Marshal(f.Patch)
			if err != nil {
				return nil, fmt.Errorf("fork %q: patch: %w", f.Workload, err)
			}
			customizations.Patch = &models.SandboxCustomPatch{
				Type:  models.SandboxesPatchTypeStrategic,
				Value: string(d),
			}
		}

		fork := &models.SandboxFork{
			ForkOf: &models.SandboxForkOf{
				Kind:      &kind,
				Name:      &f.Workload,
				Namespace: &ns,
			},
			Customizations: customizations,
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
			Cluster:     &vals.Cluster,
			Description: vals.Description,
			Forks:       forks,
			Resources:   resources,
		},
	}
	if len(vals.Labels) > 0 {
		sb.Spec.Labels = copyMap(vals.Labels)
	}
	if vals.TTL != "" {
		sb.Spec.TTL = &models.SandboxTTL{Duration: vals.TTL}
	}
	if len(vals.DefaultRouteGroup) > 0 {
		endpoints := make([]*models.EndpointsEndpoint, 0, len(vals.DefaultRouteGroup))
		for _, e := range vals.DefaultRouteGroup {
			endpoints = append(endpoints, &models.EndpointsEndpoint{Name: e.Name, Target: e.Target})
		}
		sb.Spec.DefaultRouteGroup = &models.SandboxDefaultRouteGroup{Endpoints: endpoints}
	}
	return sb, nil
}

// resolveImages implements the per-fork image resolution precedence:
// images > image > defaults.imageTemplate > none.
func resolveImages(f *ValuesFork, imgTpl string, ctx CIContext, ns string) ([]*models.SandboxImage, error) {
	if len(f.Images) > 0 {
		out := make([]*models.SandboxImage, 0, len(f.Images))
		for i := range f.Images {
			img := &f.Images[i]
			if img.Image == "" {
				return nil, fmt.Errorf("fork %q: images[%d]: image is required", f.Workload, i)
			}
			out = append(out, &models.SandboxImage{Image: img.Image, Container: img.Container})
		}
		return out, nil
	}
	if f.Image != "" {
		return []*models.SandboxImage{{Image: f.Image}}, nil
	}
	if imgTpl != "" {
		img, err := ResolveImageTemplate(imgTpl, ImagePlaceholders(ctx, f.Workload, ns))
		if err != nil {
			return nil, fmt.Errorf("fork %q: %w", f.Workload, err)
		}
		return []*models.SandboxImage{{Image: img}}, nil
	}
	// A fork that only overrides env, or only adds an endpoint, is legal.
	return nil, nil
}

// ResolveImageTemplate substitutes {placeholder} tokens, failing (and naming the
// placeholder) when a referenced value is unavailable — an unresolved
// placeholder would otherwise reach the cluster as a literal and fail to pull.
func ResolveImageTemplate(tpl string, ph map[string]string) (string, error) {
	var missing []string
	seen := map[string]bool{}
	out := imagePlaceholderRx.ReplaceAllStringFunc(tpl, func(tok string) string {
		key := tok[1 : len(tok)-1]
		v := ph[key]
		if v == "" {
			if !seen[key] {
				seen[key] = true
				missing = append(missing, key)
			}
			return tok
		}
		return v
	})
	if len(missing) > 0 {
		return "", fmt.Errorf("unresolvable image-template placeholder(s): %s",
			strings.Join(missing, ", "))
	}
	return out, nil
}

// mergeEnv merges defaults.env under per-fork env (per-fork wins) and compiles
// each value to a sandbox env var, resolving fromResource / ${resource:...} refs.
func mergeEnv(defEnv, forkEnv map[string]EnvValue, declared map[string]bool, workload string) ([]*models.SandboxEnvVar, error) {
	merged := make(map[string]EnvValue, len(defEnv)+len(forkEnv))
	for k, v := range defEnv {
		merged[k] = v
	}
	for k, v := range forkEnv {
		merged[k] = v
	}
	if len(merged) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	// Sorted so a rendered spec is identical between runs, which is what makes
	// dry-run output diffable.
	sort.Strings(keys)

	out := make([]*models.SandboxEnvVar, 0, len(keys))
	for _, k := range keys {
		v := merged[k]
		ev := &models.SandboxEnvVar{Name: k}
		ref := ""
		switch {
		case v.IsRef:
			ref = v.FromResource
		default:
			if m := fullResourceSigilRx.FindStringSubmatch(v.Value); m != nil {
				ref = m[1]
			}
		}
		if ref != "" {
			resolved, err := resourceRef(ref, declared, k, workload)
			if err != nil {
				return nil, err
			}
			ev.ValueFrom = &models.SandboxEnvValueFrom{Resource: resolved}
			out = append(out, ev)
			continue
		}
		// Unescape $${...} -> ${...}, so a value that genuinely wants those
		// characters can say so.
		ev.Value = strings.ReplaceAll(v.Value, "$${", "${")
		out = append(out, ev)
	}
	return out, nil
}

// resourceRef parses a "name.key" reference and checks that the resource is
// declared, since an undeclared one would otherwise fail late and obscurely.
func resourceRef(ref string, declared map[string]bool, envKey, workload string) (*models.SandboxEnvValueFromResource, error) {
	prefix := fmt.Sprintf("fork %q: env %q: ", workload, envKey)
	name, key, found := strings.Cut(ref, ".")
	if !found || name == "" || key == "" {
		return nil, fmt.Errorf("%sresource reference %q must be of the form name.key", prefix, ref)
	}
	if !declared[name] {
		return nil, fmt.Errorf("%sresource reference %q names an undeclared resource %q",
			prefix, ref, name)
	}
	return &models.SandboxEnvValueFromResource{Name: name, OutputKey: key}, nil
}

func copyMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
