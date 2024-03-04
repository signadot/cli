package resourceplugin

import (
	"encoding/json"

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
	name, spec, err := utils.UnstructuredToNameAndSpec(un)
	if err != nil {
		return nil, err
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
