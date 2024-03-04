package resourceplugin

import (
	"encoding/json"
	"errors"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/jsonexact"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
)

func loadResourcePlugin(file string, tplVals config.TemplateVals, forDelete bool) (*models.ResourcePlugin, error) {
	template, err := utils.LoadUnstructuredTemplate(file, tplVals, forDelete)
	if err != nil {
		return nil, err
	}
	return unstructuredToResourcePlugin(template)
}

func unstructuredToResourcePlugin(un any) (*models.ResourcePlugin, error) {
	var (
		name string
		ok   bool
		spec any
	)
	switch x := un.(type) {
	case map[string]any:
		name, ok = x["name"].(string)
		spec = x["spec"]
	default:
	}
	if !ok {
		return nil, errors.New("missing name or spec fields")
	}
	d, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	rp := &models.ResourcePlugin{Name: name}
	if err := jsonexact.Unmarshal(d, &rp.Spec); err != nil {
		return nil, err
	}
	return rp, nil
}
