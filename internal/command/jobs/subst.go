package jobs

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/jsonexact"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
)

func loadJob(file string, tplVals config.TemplateVals, forDelete bool) (*models.Job, error) {
	template, err := utils.LoadUnstructuredTemplate(file, tplVals, forDelete)
	if err != nil {
		return nil, err
	}

	return unstructuredToJob(template)
}

func unstructuredToJob(un any) (*models.Job, error) {
	raw, ok := un.(map[string]any)
	if !ok {
		return nil, errors.New("missing spec field")
	}
	spec := raw["spec"]

	d, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	rg := &models.Job{}
	if err := jsonexact.Unmarshal(d, &rg.Spec); err != nil {
		return nil, fmt.Errorf("couldn't parse YAML job definition - %s",
			strings.TrimPrefix(err.Error(), "json: "))
	}
	return rg, nil
}
