package sandbox

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/signadot/cli/internal/clio"
	"github.com/signadot/go-sdk/models"
)

func loadSandbox(file string, args []string) (*models.Sandbox, error) {
	substMap, err := substMap(args)
	if err != nil {
		return nil, err
	}

	sbt, err := clio.LoadYAML[any](file)
	if err != nil {
		return nil, err
	}
	if err := substTemplate(sbt, substMap); err != nil {
		return nil, err
	}
	return unstructuredToSandbox(*sbt)
}

func substMap(args []string) (map[string]string, error) {
	substMap := map[string]string{}
	for _, arg := range args {
		varName, val, found := strings.Cut(arg, "=")
		if !found {
			return nil, fmt.Errorf("arg %q is not in <var>=<value> form", arg)
		}
		if err := checkVar(varName); err != nil {
			return nil, fmt.Errorf("arg %q has invalid variable %q", arg, varName)
		}
		substMap[varName] = val
	}
	return substMap, nil
}

func substTemplate(sbt *any, substMap map[string]string) error {
	vars := map[string]struct{}{}
	err := substTemplateRec(sbt, substMap, vars)
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
		return fmt.Errorf("unexpanded variables: %s", strings.Join(notExpanded, ", "))
	}
	return nil
}

func substTemplateRec(sbt *any, substMap map[string]string, vars map[string]struct{}) error {
	switch x := (*sbt).(type) {
	case map[string]any:
		for k, v := range x {
			if err := substTemplateRec(&v, substMap, vars); err != nil {
				return err
			}
			x[k] = v
		}

	case []any:
		for _, v := range x {
			if err := substTemplateRec(&v, substMap, vars); err != nil {
				return err
			}
		}
	case string:
		*sbt = substString(x, substMap, vars)
	default:
	}
	return nil
}

var varRefRx = regexp.MustCompile(`\$\{([a-zA-Z][a-zA-Z0-9_.-]*)\}`)

func substString(s string, substMap map[string]string, vars map[string]struct{}) string {
	matches := varRefRx.FindAllStringSubmatchIndex(s, -1)
	if matches == nil {
		return s
	}
	result := []string{}
	cur, start, end := 0, 0, 0
	for i := range matches {
		// begin and end of submatch corresponding to variable name
		// in ${<var-name>}.
		start, end = matches[i][2], matches[i][3]
		// store any skipped string
		if cur < start-2 {
			result = append(result, s[cur:start-2]) // ${
		}
		v := s[start:end]
		end++ // }
		cur = end
		vars[v] = struct{}{}
		repl, ok := substMap[v]
		if !ok {
			// unsubstituted variables are handled
			// in substTemplate to report all of them
			// no error is reported here.
			continue
		}
		result = append(result, repl)
	}
	if end < len(s) {
		result = append(result, s[end:])
	}
	return strings.Join(result, "")
}

func unstructuredToSandbox(un any) (*models.Sandbox, error) {
	if err := port2Int(&un); err != nil {
		return nil, err
	}
	d, err := json.Marshal(un)
	if err != nil {
		return nil, err
	}
	var sb models.Sandbox
	if err := json.Unmarshal(d, &sb); err != nil {
		return nil, err
	}
	return &sb, nil
}

// translates all port values to ints if they are strings.
func port2Int(un *any) error {
	switch x := (*un).(type) {
	case map[string]any:
		for k, v := range x {
			if k != "port" {
				if err := port2Int(&v); err != nil {
					return err
				}
				x[k] = v
				continue
			}
			ps, ok := v.(string)
			if !ok {
				continue
			}
			p, err := strconv.ParseInt(ps, 10, 32)
			if err != nil {
				return fmt.Errorf("port is not int: %q", ps)
			}
			x[k] = p
		}
	case []any:
		for i := range x {
			if err := port2Int(&x[i]); err != nil {
				return err
			}
		}
	default:
	}
	return nil
}

var varPat = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_.-]*$`)

func checkVar(varName string) error {
	if !varPat.MatchString(varName) {
		return fmt.Errorf("invalid variable name %q, should match %s", varName, varPat)
	}
	return nil
}
