package planrunnergroup

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/jsonexact"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
)

func loadPlanRunnerGroup(file string, tplVals config.TemplateVals, forDelete bool) (*models.PlanRunnerGroup, error) {
	template, err := utils.LoadUnstructuredTemplate(file, tplVals, forDelete)
	if err != nil {
		return nil, err
	}
	return unstructuredToPlanRunnerGroup(template)
}

func unstructuredToPlanRunnerGroup(un any) (*models.PlanRunnerGroup, error) {
	name, spec, err := utils.UnstructuredToNameAndSpec(un)
	if err != nil {
		return nil, err
	}
	d, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	prg := &models.PlanRunnerGroup{Name: name}
	if err := jsonexact.Unmarshal(d, &prg.Spec); err != nil {
		return nil, fmt.Errorf("couldn't parse YAML plan runner group definition - %s",
			strings.TrimPrefix(err.Error(), "json: "))
	}
	return prg, nil
}
