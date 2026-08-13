package sandbox

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/signadot/cli/internal/clio"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/render"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/models"
)

// renderSandbox produces the sandbox to apply from -f and the flags around it,
// returning both the document, for printing, and the decoded sandbox, for
// applying. It contacts nothing, so `--dry-run=client` needs no credentials.
func renderSandbox(cfg *config.SandboxApply) (map[string]any, *models.Sandbox, error) {
	ciCtx, err := render.Detect(cfg.CIContext, render.OSEnv)
	if err != nil {
		return nil, nil, err
	}

	raw, err := clio.ReadFileOrStdin(cfg.Filename)
	if err != nil {
		return nil, nil, err
	}
	un, err := utils.RenderTemplate(raw, cfg.TemplateVals, utils.TemplateOptions{
		BaseDir: templateBaseDir(cfg.Filename),
	})
	if err != nil {
		return nil, nil, err
	}
	if err := port2Int(&un); err != nil {
		return nil, nil, err
	}

	var doc map[string]any
	if isValuesDoc(un) {
		vals, err := valuesFromUnstructured(un)
		if err != nil {
			return nil, nil, err
		}
		doc, err = render.RenderValues(vals, ciCtx, render.Options{
			Name:          cfg.Name,
			TTL:           cfg.TTL,
			DefaultLabels: cfg.DefaultLabels,
		})
		if err != nil {
			return nil, nil, err
		}
	} else {
		// Decoding a spec requires a name, but a spec written for CI can
		// legitimately leave it to --name or to the CI context, so resolve it
		// before decoding rather than rejecting the document.
		if err := resolveDocName(un, cfg.Name, ciCtx); err != nil {
			return nil, nil, err
		}
		sb, err := unstructuredToSandbox(un)
		if err != nil {
			return nil, nil, err
		}
		if err := render.ApplyContext(sb, ciCtx, "", cfg.TTL, cfg.DefaultLabels); err != nil {
			return nil, nil, err
		}
		if doc, err = render.SandboxToDoc(sb); err != nil {
			return nil, nil, err
		}
	}

	if cfg.Patch != "" {
		patch, err := loadPatch(cfg.Patch)
		if err != nil {
			return nil, nil, err
		}
		// Last, so a patch can reach anything rendering produced, including the
		// name and the labels.
		doc = render.MergePatch(doc, patch)
	}

	// Decode the finished document the way an ordinary spec is decoded, so that
	// a field a patch misspelled is caught here and not by the API.
	req, err := unstructuredToSandbox(doc)
	if err != nil {
		return nil, nil, err
	}
	return doc, req, nil
}

// isValuesDoc distinguishes the two kinds of document -f accepts. A sandbox spec
// always has a top-level `spec`, and the values schema has no such field, so the
// presence of that key decides it without needing a version marker.
func isValuesDoc(un any) bool {
	m, ok := un.(map[string]any)
	if !ok {
		return false
	}
	_, hasSpec := m["spec"]
	return !hasSpec
}

// resolveDocName settles a spec document's name before it is decoded, since
// decoding requires one and a spec written for CI can legitimately leave the name
// to --name or to the CI context.
func resolveDocName(un any, explicitName string, c render.CIContext) error {
	m, ok := un.(map[string]any)
	if !ok {
		return nil
	}
	var docName string
	if v, present := m["name"]; present && v != nil {
		s, ok := v.(string)
		if !ok {
			// Leave a malformed name for the decoder to report.
			return nil
		}
		docName = s
	}
	name, err := render.ResolveName(docName, explicitName, c)
	if err != nil {
		return err
	}
	m["name"] = name
	return nil
}

// valuesFromUnstructured re-decodes a rendered document into the values schema,
// which is where unknown fields are rejected.
func valuesFromUnstructured(un any) (*render.Values, error) {
	d, err := json.Marshal(un)
	if err != nil {
		return nil, err
	}
	return render.ParseValues(d)
}

func loadPatch(file string) (map[string]any, error) {
	d, err := clio.ReadFileOrStdin(file)
	if err != nil {
		return nil, err
	}
	return render.ParseMapping(d, "the patch")
}

// templateBaseDir is the directory `@{embed: ...}` paths resolve against: the
// file's own directory, or the working directory when reading stdin.
func templateBaseDir(file string) string {
	if file == "-" {
		return "."
	}
	abs, err := filepath.Abs(file)
	if err != nil {
		return filepath.Dir(file)
	}
	return filepath.Dir(abs)
}

// writeRenderedSpec prints the rendered spec for --dry-run. YAML is the default
// because the output's job is to be read, diffed, and fed back to -f.
func writeRenderedSpec(cfg *config.SandboxApply, out io.Writer, doc map[string]any) error {
	switch cfg.OutputFormat {
	case config.OutputFormatDefault, config.OutputFormatYAML:
		y, err := render.ToYAML(doc)
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(out, y)
		return err
	case config.OutputFormatJSON:
		return print.RawJSON(out, doc)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}
