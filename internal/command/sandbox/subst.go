package sandbox

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/jsonexact"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
)

func loadSandbox(file string, tplVals config.TemplateVals, forDelete bool) (*models.Sandbox, error) {
	template, err := utils.LoadUnstructuredTemplate(file, tplVals, forDelete)
	if err != nil {
		return nil, err
	}
	return unstructuredToSandbox(template)
}

func unstructuredToSandbox(un any) (*models.Sandbox, error) {
	if err := port2Int(&un); err != nil {
		return nil, err
	}
	name, spec, err := utils.UnstructuredToNameAndSpec(un)
	if err != nil {
		return nil, err
	}
	d, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	sb := &models.Sandbox{Name: name}
	if err := jsonexact.Unmarshal(d, &sb.Spec); err != nil {
		return nil, fmt.Errorf("couldn't parse YAML sandbox definition - %s",
			strings.TrimPrefix(err.Error(), "json: "))
	}
	return sb, nil
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
