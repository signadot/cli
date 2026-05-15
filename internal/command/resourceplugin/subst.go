package resourceplugin

import (
	"encoding/json"
	"strings"

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
	rawName, spec, err := utils.UnstructuredToNameAndSpec(un)
	if err != nil {
		return nil, err
	}
	d, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	// The YAML `name:` field carries the wire form directly — either a
	// bare name (publishes the default 0.0.0) or "bareName@semver". Any
	// top-level `version:` field is ignored (the parser is non-strict);
	// users on the older two-form syntax should fold the version into
	// the name suffix.
	rp := &models.ResourcePlugin{Name: rawName}
	if err := jsonexact.Unmarshal(d, &rp.Spec); err != nil {
		return nil, err
	}
	return rp, nil
}

// splitNameVersion parses "name[@version]" into its parts. An empty version
// component is returned as "", which the server interprets as the default
// (0.0.0) on writes and as "latest" on reads.
func splitNameVersion(ref string) (name, version string) {
	if i := strings.IndexByte(ref, '@'); i >= 0 {
		return ref[:i], ref[i+1:]
	}
	return ref, ""
}

// formatNameRef joins a name and version back into "name@version", or returns
// the bare name when the version is empty.
func formatNameRef(name, version string) string {
	if version == "" {
		return name
	}
	return name + "@" + version
}
