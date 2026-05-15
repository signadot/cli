package resourceplugin

import (
	"encoding/json"
	"fmt"
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
	name, suffixVersion := splitNameVersion(rawName)
	topVersion, err := optionalStringField(un, "version")
	if err != nil {
		return nil, err
	}
	// Reject the version being supplied in both forms — even when the values
	// match — so the spec has a single source of truth.
	if suffixVersion != "" && topVersion != "" {
		return nil, fmt.Errorf("version is set in both 'name' (%q) and the top-level 'version' field (%q); pick one", rawName, topVersion)
	}
	version := suffixVersion
	if version == "" {
		version = topVersion
	}
	// The wire model carries a single `name` field of the form
	// "bareName[@semver]"; the CLI's two-form YAML (suffix-on-name or
	// top-level `version:`) is collapsed here before the rp is sent.
	rp := &models.ResourcePlugin{Name: formatNameRef(name, version)}
	if err := jsonexact.Unmarshal(d, &rp.Spec); err != nil {
		return nil, err
	}
	return rp, nil
}

// optionalStringField returns the named top-level field as a string, or "" if
// absent. It errors if the field is present but not a string so a YAML typo
// like `version: 1.2.0` (parsed as a number by some parsers) fails loudly
// rather than being silently treated as missing.
func optionalStringField(un any, key string) (string, error) {
	m, ok := un.(map[string]any)
	if !ok {
		return "", nil
	}
	v, present := m[key]
	if !present {
		return "", nil
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%q field must be a string, got %T", key, v)
	}
	return s, nil
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
