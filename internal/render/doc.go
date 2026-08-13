package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/signadot/go-sdk/models"
	goyaml "sigs.k8s.io/yaml/goyaml.v2"
)

// SandboxToDoc converts a typed sandbox into the plain document that gets
// serialised or patched.
//
// Nulls are pruned because several fields in the API models lack `omitempty`,
// so a straight marshal produces a spec littered with `local: null` and the
// like. That noise would dominate dry-run output and make it hard to diff.
func SandboxToDoc(sb *models.Sandbox) (map[string]any, error) {
	d, err := json.Marshal(sb)
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	if err := json.Unmarshal(d, &doc); err != nil {
		return nil, err
	}
	pruned, ok := PruneNulls(doc).(map[string]any)
	if !ok {
		return nil, fmt.Errorf("rendered sandbox is not a mapping")
	}
	return pruned, nil
}

// MergePatch applies an RFC-7386-style merge patch to a document: maps merge
// recursively, scalars and arrays replace wholesale, and an explicit null
// deletes the key.
func MergePatch(dst, patch map[string]any) map[string]any {
	for k, pv := range patch {
		if pv == nil {
			delete(dst, k)
			continue
		}
		pm, pOK := pv.(map[string]any)
		dm, dOK := dst[k].(map[string]any)
		if pOK && dOK {
			dst[k] = MergePatch(dm, pm)
			continue
		}
		dst[k] = pv
	}
	return dst
}

// PruneNulls removes null-valued keys, recursively.
func PruneNulls(v any) any {
	switch x := v.(type) {
	case []any:
		out := make([]any, 0, len(x))
		for _, item := range x {
			out = append(out, PruneNulls(item))
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			if val == nil {
				continue
			}
			out[k] = PruneNulls(val)
		}
		return out
	default:
		return v
	}
}

// keyOrder lists, per document path, the keys that should lead a mapping.
// Anything unlisted follows in alphabetical order. This puts the short fields
// that identify a sandbox above the bulky nested ones, matching how specs are
// written in the Signadot documentation — labels in particular are worth seeing
// without scrolling past every fork.
var keyOrder = map[string][]string{
	"":     {"name", "spec"},
	"spec": {"cluster", "description", "labels", "ttl"},
}

// ToYAML serialises a document with keys ordered by keyOrder rather than purely
// alphabetically, and with sequences unindented.
func ToYAML(doc map[string]any) (string, error) {
	// Round-trip through JSON the way sigs.k8s.io/yaml does internally, but with
	// UseNumber, so whole numbers stay integers rather than becoming floats and
	// a port does not render as 8080.
	d, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	dec := json.NewDecoder(bytes.NewReader(d))
	dec.UseNumber()
	var normalised any
	if err := dec.Decode(&normalised); err != nil {
		return "", err
	}

	out, err := goyaml.Marshal(ordered(normalised, ""))
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// ordered rewrites maps as MapSlices so the emitter keeps our key order instead
// of imposing its own, and resolves the json.Numbers left by the decoder above.
func ordered(v any, path string) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(goyaml.MapSlice, 0, len(t))
		for _, k := range orderKeys(t, path) {
			out = append(out, goyaml.MapItem{Key: k, Value: ordered(t[k], childPath(path, k))})
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = ordered(e, path+"[]")
		}
		return out
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return i
		}
		if f, err := t.Float64(); err == nil {
			return f
		}
		return t.String()
	default:
		return v
	}
}

func orderKeys(m map[string]any, path string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	preferred, ok := keyOrder[path]
	if !ok {
		return keys
	}
	rank := make(map[string]int, len(preferred))
	for i, k := range preferred {
		rank[k] = i
	}
	// Stable, over an already alphabetical slice, so unranked keys keep that
	// order behind the ranked ones.
	sort.SliceStable(keys, func(i, j int) bool {
		ri, iRanked := rank[keys[i]]
		rj, jRanked := rank[keys[j]]
		if iRanked && jRanked {
			return ri < rj
		}
		return iRanked && !jRanked
	})
	return keys
}

func childPath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

// ValidateSandbox catches the mistakes that would otherwise surface as an opaque
// API rejection after the sandbox has already been submitted.
func ValidateSandbox(sb *models.Sandbox) error {
	if sb.Name == "" {
		return fmt.Errorf("rendered sandbox has no name")
	}
	if len(sb.Name) > MaxNameLen {
		return fmt.Errorf("sandbox name %q exceeds %d characters", sb.Name, MaxNameLen)
	}
	if sb.Spec == nil || sb.Spec.Cluster == nil || *sb.Spec.Cluster == "" {
		return fmt.Errorf("rendered sandbox spec must specify a cluster")
	}
	for i, f := range sb.Spec.Forks {
		if f.ForkOf == nil || f.ForkOf.Name == nil || *f.ForkOf.Name == "" {
			return fmt.Errorf("forks[%d]: forkOf.name is required", i)
		}
		if f.ForkOf.Namespace == nil || *f.ForkOf.Namespace == "" {
			return fmt.Errorf("forks[%d]: forkOf.namespace is required", i)
		}
	}
	return nil
}
