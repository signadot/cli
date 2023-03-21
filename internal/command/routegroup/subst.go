package routegroup

import (
	"encoding/json"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
)

func loadRouteGroup(file string, tplVals config.TemplateVals, forDelete bool) (*models.RouteGroup, error) {
	template, err := utils.LoadUnstructuredTemplate(file, tplVals, forDelete)
	if err != nil {
		return nil, err
	}
	return unstructuredToRouteGroup(template)
}

func unstructuredToRouteGroup(un any) (*models.RouteGroup, error) {
	d, err := json.Marshal(un)
	if err != nil {
		return nil, err
	}
	var rg models.RouteGroup
	if err := json.Unmarshal(d, &rg); err != nil {
		return nil, err
	}
	return &rg, nil
}
