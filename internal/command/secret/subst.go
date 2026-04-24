package secret

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/jsonexact"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
)

// loadSecretFile reads a flat secret YAML/JSON file with `--set` expansion.
// Unlike resource plugins the file has no `spec:` stanza; fields map 1:1 to the Secret model.
// When forDelete is true, only the `name` field is retained after substitution.
func loadSecretFile(file string, tplVals config.TemplateVals, forDelete bool) (*models.Secret, error) {
	template, err := utils.LoadUnstructuredTemplate(file, tplVals, forDelete)
	if err != nil {
		return nil, err
	}
	if _, ok := template.(map[string]any); !ok {
		return nil, fmt.Errorf("secret file must be a YAML/JSON object")
	}
	d, err := json.Marshal(template)
	if err != nil {
		return nil, err
	}
	s := &models.Secret{}
	if err := jsonexact.Unmarshal(d, s); err != nil {
		return nil, fmt.Errorf("couldn't parse secret file - %s",
			strings.TrimPrefix(err.Error(), "json: "))
	}
	return s, nil
}
