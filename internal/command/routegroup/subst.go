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
	name, spec, err := utils.UnstructuredToNameAndSpec(un)
	if err != nil {
		return nil, err
	}
	d, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	rg := &models.RouteGroup{Name: name}
	if err := jsonexact.Unmarshal(d, &rg.Spec); err != nil {
		return nil, fmt.Errorf("couldn't parse YAML routegroup definition - %s",
			strings.TrimPrefix(err.Error(), "json: "))
	}
	return rg, nil
}
