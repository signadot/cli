package resourceplugin

import (
	"encoding/json"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
)

func loadResourcePlugin(file string, tplVals config.TemplateVals, forDelete bool) (*models.ResourcePlugin, error) {
	template, err := utils.LoadUnstructuredTemplate(file, tplVals, forDelete, utils.ReadFileContent)
	if err != nil {
		return nil, err
	}
	return unstructuredToResourcePlugin(template)
}

func unstructuredToResourcePlugin(un any) (*models.ResourcePlugin, error) {
	d, err := json.Marshal(un)
	if err != nil {
		return nil, err
	}
	var rp models.ResourcePlugin
	if err := json.Unmarshal(d, &rp); err != nil {
		return nil, err
	}
	return &rp, nil
}
