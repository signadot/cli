package utils

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"sigs.k8s.io/yaml"

	"github.com/signadot/cli/internal/clio"
	"github.com/signadot/cli/internal/config"
)

var (
	placeholderPattern = regexp.MustCompile(`@{[^}]+}`)

	errUnexpandedVar = errors.New("unexpanded variable")
	errUnsupportedOp = errors.New("unsupported operation")
	errInvalidEnc    = errors.New("invalid encoding")
	errInvalidVar    = errors.New("invalid variable name")
)

func LoadUnstructuredTemplate(file string, tplVals config.TemplateVals, forDelete bool) (any, error) {
	substMap, err := substMap(tplVals)
	if err != nil {
		return nil, err
	}
	template, err := clio.LoadYAML[any](file)
	if err != nil {
		return nil, err
	}
	if forDelete {
		*template = extractName(*template)
	}
	if err := substTemplate(template, substMap, file); err != nil {
		return nil, err
	}
	return *template, nil
}

func UnstructuredToNameAndSpec(un any) (name string, spec any, err error) {
	var ok bool
	switch x := un.(type) {
	case map[string]any:
		name, ok = x["name"].(string)
		spec = x["spec"]
	default:
	}
	if !ok {
		err = errors.New("missing name or spec fields")
		return "", nil, err
	}
	return
}

func extractName(rpt any) map[string]any {
	topLevel, ok := rpt.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	for k := range topLevel {
		if k == "name" {
			continue
		}
		delete(topLevel, k)
	}
	return topLevel
}

// does template substitution, including error checking
// that variables in the template are expanded
func substTemplate(rpt *any, substMap map[string]string, templatePath string) error {
	absPath, err := filepath.Abs(templatePath)
	if err != nil {
		return err
	}
	wdir := filepath.Dir(absPath)
	vars := map[string]struct{}{}
	err = substTemplateRec(rpt, substMap, vars, wdir)
	if err != nil {
		return err
	}
	notExpanded := []string{}
	for k := range vars {
		if _, ok := substMap[k]; !ok {
			notExpanded = append(notExpanded, k)
		}
	}
	if len(notExpanded) > 0 {
		return fmt.Errorf("%w: %s", errUnexpandedVar, strings.Join(notExpanded, ", "))
	}
	return nil
}

// returns a map[string]string of variable names to values from
// cli template vals, checks for conflicts
func substMap(tplVals []config.TemplateVal) (map[string]string, error) {
	substMap := map[string]string{}
	conflicts := map[string][]string{}
	for _, tv := range tplVals {
		if tVal, present := substMap[tv.Var]; present {
			if tVal != tv.Val {
				if len(conflicts[tv.Var]) == 0 {
					conflicts[tv.Var] = []string{tVal}
				}
				conflicts[tv.Var] = append(conflicts[tv.Var], tv.Val)
				continue
			}
		}
		substMap[tv.Var] = tv.Val
	}
	if len(conflicts) == 0 {
		return substMap, nil
	}
	conflictKeys := make([]string, 0, len(conflicts))
	for k := range conflicts {
		conflictKeys = append(conflictKeys, k)
	}
	sort.Strings(conflictKeys)
	msgs := make([]string, 0, len(conflictKeys))
	for _, key := range conflictKeys {
		vals := strings.Join(conflicts[key], ", ")
		msgs = append(msgs, fmt.Sprintf("%s: %s", key, vals))
	}
	return nil, fmt.Errorf("conflicting variable defs:\n\t%s", strings.Join(msgs, "\n\t"))
}

// actually substitutes a yaml template
func substTemplateRec(rpt *any, substMap map[string]string, vars map[string]struct{}, wdir string) error {
	switch x := (*rpt).(type) {
	case map[string]any:
		for k, v := range x {
			if err := substTemplateRec(&v, substMap, vars, wdir); err != nil {
				return err
			}
			x[k] = v
		}

	case []any:
		for i := range x {
			if err := substTemplateRec(&x[i], substMap, vars, wdir); err != nil {
				return err
			}
		}
	case string:
		res, err := substString(x, substMap, vars, wdir)
		if err != nil {
			return err
		}
		*rpt = res
	default:
	}
	return nil
}

// substitution of or within a yaml string
func substString(s string, substMap map[string]string, vars map[string]struct{}, wd string) (any, error) {
	matches := placeholderPattern.FindAllStringIndex(s, -1)
	if matches == nil {
		return s, nil
	}
	// check if s is exactly @{...}.  If it is
	// then we can embed in ways other than raw.
	if len(matches) == 1 {
		begin, end := matches[0][0], matches[0][1]
		if begin == 0 && end == len(s) {
			data, encType, err := getValue(substMap, vars, getReplSpec(s, 0, len(s)), wd)
			if err != nil {
				return nil, err
			}
			enc, err := encodeValue(data, encType)
			if err != nil {
				return nil, err
			}
			return enc, nil
		}
	}

	// proper string interpolation, we need to slice and dice s
	// into parts.  parts will contain segments of s that are
	// not in @{...} interleaved with substitutions for
	// whatever is found in @{...} (raw embedding)
	parts := make([]string, 0, len(matches)*2+1)
	var (
		begin, end, lastEnd int
	)

	for i := range matches {
		match := matches[i]
		begin, end = match[0], match[1]
		parts = append(parts, s[lastEnd:begin])
		lastEnd = end // }
		subst, ty, err := getValue(substMap, vars, getReplSpec(s, begin, end), wd)
		if err != nil {
			return "", err
		}
		if ty != opRaw {
			return "", fmt.Errorf("%w: embed %s does not work as part of a larger string", errInvalidEnc, ty)
		}
		parts = append(parts, string(subst))
	}
	if lastEnd < len(s) {
		parts = append(parts, s[lastEnd:])
	}
	return strings.Join(parts, ""), nil
}

func getValue(substMap map[string]string, vars map[string]struct{}, replSpec, wdir string) ([]byte, opType, error) {
	replSpec = strings.TrimSpace(replSpec)
	opSpec, rest, found := strings.Cut(replSpec, ":")
	if !found {
		// variable
		varRef := getOp(replSpec)
		if !config.VarRx.MatchString(varRef) {
			return nil, 0, fmt.Errorf("%w: %q", errInvalidVar, replSpec)
		}
		ty, err := getOpType(replSpec)
		vars[varRef] = struct{}{}
		if err != nil {
			return nil, 0, err
		}
		return []byte(substMap[varRef]), ty, nil
	}
	// directive
	opSpec = strings.TrimSpace(opSpec)
	rest = strings.TrimSpace(rest)
	ty, err := getOpType(opSpec)
	if err != nil {
		return nil, 0, fmt.Errorf("error parsing %q: %w", opSpec, err)
	}
	switch getOp(opSpec) {
	case "embed":
		p := filepath.Join(wdir, rest)
		d, e := os.ReadFile(p)
		if e != nil {
			return nil, 0, e
		}
		return d, ty, nil
	default:
		return nil, 0, fmt.Errorf("error parsing template: %w: %q", errUnsupportedOp, opSpec)
	}
}

// getReplSpec returns the part inside @{...}
// given match indices begin, end from a match placeholderPattern
// on s.
func getReplSpec(s string, begin, end int) string {
	return s[begin+2 : end-1]
}

func encodeValue(d []byte, t opType) (any, error) {
	switch t {
	case opRaw:
		return string(d), nil
	case opYaml:
		var a any
		err := yaml.Unmarshal(d, &a)
		if err != nil {
			return "", err
		}
		return a, nil
	case opBinary:
		return base64.StdEncoding.EncodeToString(d), nil
	default:
		panic(t)
	}
}

func getOp(opSpec string) string {
	lsb := strings.IndexByte(opSpec, '[')
	if lsb == -1 {
		return opSpec
	}
	return opSpec[:lsb]
}

type opType int

const (
	opRaw    opType = iota
	opYaml          = iota
	opBinary        = iota
	// TODO add opTemplate for embedding expanded templates
)

func (t opType) String() string {
	switch t {
	case opRaw:
		return "raw"
	case opYaml:
		return "yaml"
	case opBinary:
		return "binary"
	default:
		panic(fmt.Sprintf("unknown type <%d>", t))
	}
}

func getOpType(opSpec string) (opType, error) {
	lsb := strings.IndexByte(opSpec, '[')
	if lsb == -1 {
		// default value
		return opRaw, nil
	}
	rsb := strings.LastIndexByte(opSpec, ']')
	if rsb == -1 {
		return 0, fmt.Errorf("invalid op spec: %q imbalanced '[]'", opSpec)
	}
	tySpec := strings.TrimSpace(opSpec[lsb+1 : rsb])
	switch tySpec {
	case "raw":
		return opRaw, nil
	case "yaml":
		return opYaml, nil
	case "binary":
		return opBinary, nil
	default:
		return 0, fmt.Errorf("unrecognized template op type: %q", tySpec)
	}
}
