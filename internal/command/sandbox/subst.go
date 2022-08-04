package sandbox

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/signadot/cli/internal/clio"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/models"
)

func loadSandbox(file string, tplVals config.TemplateVals) (*models.Sandbox, error) {
	substMap, err := substMap(tplVals)
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
	return nil, fmt.Errorf("conflicting variable defs:\n\t%s\n", strings.Join(msgs, "\n\t"))
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
		for i := range x {
			if err := substTemplateRec(&x[i], substMap, vars); err != nil {
				return err
			}
		}
	case string:
		*sbt = substString(x, substMap, vars)
	default:
	}
	return nil
}

func substString(s string, substMap map[string]string, vars map[string]struct{}) string {
	matches := config.VarRefRx.FindAllStringSubmatchIndex(s, -1)
	if matches == nil {
		return s
	}
	result := []string{}
	cur, start, end := 0, 0, 0
	for i := range matches {
		// begin and end of submatch corresponding to variable name
		// in @{<var-name>}.
		start, end = matches[i][2], matches[i][3]
		// store any skipped string
		if cur < start-2 {
			result = append(result, s[cur:start-2]) // @{
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
			port, err := strconv.ParseInt(ps, 10, 32)
			if err != nil {
				return fmt.Errorf("port is not int: %q", ps)
			}
			x[k] = port
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
