package sandbox

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
)

func loadSandbox(file string, tplVals config.TemplateVals, forDelete bool) (*models.Sandbox, error) {
	template, err := utils.LoadUnstructuredTemplate(file, tplVals, forDelete, utils.ReadFileContent)
	if err != nil {
		return nil, err
	}
	return unstructuredToSandbox(template)
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
