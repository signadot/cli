package jobrunnergroup

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/jsonexact"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
)

func loadRunnerGroup(file string, tplVals config.TemplateVals, forDelete bool) (*models.RunnergroupsRunnerGroup, error) {
	template, err := utils.LoadUnstructuredTemplate(file, tplVals, forDelete)
	if err != nil {
		return nil, err
	}
	return unstructuredToRunnerGroup(template)
}

func unstructuredToRunnerGroup(un any) (*models.RunnergroupsRunnerGroup, error) {
	name, spec, err := utils.UnstructuredToNameAndSpec(un)
	if err != nil {
		return nil, err
	}
	d, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	rg := &models.RunnergroupsRunnerGroup{Name: name}
	if err := jsonexact.Unmarshal(d, &rg.Spec); err != nil {
		return nil, fmt.Errorf("couldn't parse YAML jobrunnergroup definition - %s",
			strings.TrimPrefix(err.Error(), "json: "))
	}
	return rg, nil
}
