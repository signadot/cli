package routegroup

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/jsonexact"
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
	if err := jsonexact.Unmarshal(d, &rg); err != nil {
		return nil, fmt.Errorf("couldn't parse YAML routegroup definition - %s",
			strings.TrimPrefix(err.Error(), "json: "))
	}
	return &rg, nil
}
