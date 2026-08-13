// Package render turns a values document into the sandbox document that
// `signadot sandbox apply` submits. The values schema is the declarative,
// template-free way to describe a sandbox: callers state which workloads to
// fork and with what, and the built-in template supplies the structure.
package render

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"sigs.k8s.io/yaml"
)

// Values is the values document, version 1.
//
// It exists so that a caller who has no sandbox template of their own — most
// obviously a CI job — can describe a sandbox in the terms they already think
// in, rather than in the shape the API happens to want. Everything here is
// either flat or a short list, so a CI integration can build it from string
// inputs without a YAML library.
type Values struct {
	Name              string               `json:"name,omitempty"`
	Cluster           string               `json:"cluster,omitempty"`
	Description       string               `json:"description,omitempty"`
	Defaults          *ValuesDefaults      `json:"defaults,omitempty"`
	TTL               string               `json:"ttl,omitempty"`
	Labels            map[string]string    `json:"labels,omitempty"`
	Resources         []ValuesResource     `json:"resources,omitempty"`
	Forks             []ValuesFork         `json:"forks,omitempty"`
	DefaultRouteGroup []RouteGroupEndpoint `json:"defaultRouteGroup,omitempty"`
}

type ValuesFork struct {
	Workload  string              `json:"workload,omitempty"`
	Kind      string              `json:"kind,omitempty"`
	Namespace string              `json:"namespace,omitempty"`
	Image     string              `json:"image,omitempty"`
	Images    []ValuesImage       `json:"images,omitempty"`
	Env       map[string]EnvValue `json:"env,omitempty"`
	Endpoints []ValuesEndpoint    `json:"endpoints,omitempty"`
	Patch     map[string]any      `json:"patch,omitempty"`
}

type ValuesImage struct {
	Container string `json:"container,omitempty"`
	Image     string `json:"image,omitempty"`
}

type ValuesEndpoint struct {
	Name     string `json:"name,omitempty"`
	Port     int64  `json:"port,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

// ValuesDefaults holds the settings a caller would otherwise repeat on every
// fork. ImageTemplate is the useful one: it lets a monorepo pipeline say
// "images are tagged like this" once instead of per workload.
type ValuesDefaults struct {
	Namespace     string              `json:"namespace,omitempty"`
	ImageTemplate string              `json:"imageTemplate,omitempty"`
	Env           map[string]EnvValue `json:"env,omitempty"`
}

type ValuesResource struct {
	Name   string            `json:"name,omitempty"`
	Plugin string            `json:"plugin,omitempty"`
	Params map[string]string `json:"params,omitempty"`
}

// RouteGroupEndpoint names a service the sandbox should expose a preview URL
// for. Unlike a fork endpoint, the target is an arbitrary in-cluster address, so
// it can point at a workload this sandbox does not fork.
type RouteGroupEndpoint struct {
	Name   string `json:"name,omitempty"`
	Target string `json:"target,omitempty"`
}

// EnvValue is either a plain string or the structured {fromResource: name.key}
// form. String values may additionally embed the ${resource:name.key} sigil,
// which is the only way to reference a resource from a flat string input.
type EnvValue struct {
	Value        string
	FromResource string
	IsRef        bool
}

func StringEnv(v string) EnvValue { return EnvValue{Value: v} }

func (e *EnvValue) UnmarshalJSON(b []byte) error {
	var raw any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	switch v := raw.(type) {
	case map[string]any:
		ref, ok := v["fromResource"]
		if !ok || len(v) != 1 {
			return fmt.Errorf("env value must be a string or {fromResource: name.key}")
		}
		s, ok := ref.(string)
		if !ok {
			return fmt.Errorf("fromResource must be a string")
		}
		e.FromResource, e.IsRef = s, true
		return nil
	case []any:
		return fmt.Errorf("env value must be a string or {fromResource: name.key}")
	case nil:
		// A bare `KEY:` in YAML means an empty value.
		e.Value = ""
		return nil
	default:
		// Bare YAML scalars are accepted and stringified, since env values are
		// strings in the API and `PORT: 8080` is the natural thing to write.
		e.Value = scalarToString(v)
		return nil
	}
}

func (e EnvValue) MarshalJSON() ([]byte, error) {
	if e.IsRef {
		return json.Marshal(map[string]string{"fromResource": e.FromResource})
	}
	return json.Marshal(e.Value)
}

func scalarToString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case bool:
		return strconv.FormatBool(x)
	case float64:
		// YAML integers arrive as float64 through the JSON bridge; render them
		// without a spurious ".0" so `PORT: 8080` becomes "8080".
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'g', -1, 64)
	case json.Number:
		return x.String()
	default:
		return fmt.Sprint(x)
	}
}

// ParseValues decodes a values document, rejecting unknown fields so that a
// typo like `envs:` fails rather than being silently dropped.
func ParseValues(data []byte) (*Values, error) {
	vals := &Values{}
	if len(data) == 0 {
		return vals, nil
	}
	if err := yaml.UnmarshalStrict(data, vals); err != nil {
		return nil, fmt.Errorf("couldn't parse the values document: %s", plainDecodeError(err))
	}
	return vals, nil
}

// plainDecodeError strips the layers the YAML-to-JSON decoders wrap around what
// is usually a one-line complaint about a single field, so that the useful part
// of the message is not buried.
func plainDecodeError(err error) string {
	msg := err.Error()
	for _, prefix := range []string{
		"error unmarshaling JSON: ",
		"error converting YAML to JSON: ",
		"while decoding JSON: ",
		"json: ",
		"yaml: ",
	} {
		msg = strings.TrimPrefix(msg, prefix)
	}
	return msg
}

// ParseMapping decodes a YAML or JSON mapping, such as a merge patch.
func ParseMapping(data []byte, what string) (map[string]any, error) {
	var doc any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("couldn't parse %s: %w", what, err)
	}
	if doc == nil {
		return map[string]any{}, nil
	}
	m, ok := doc.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be a YAML or JSON mapping", what)
	}
	return m, nil
}
