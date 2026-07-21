package sandbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
	sigsyaml "sigs.k8s.io/yaml"
)

func newRender(sandbox *config.Sandbox) *cobra.Command {
	cfg := &config.SandboxRender{Sandbox: sandbox}

	cmd := &cobra.Command{
		Use:   "render",
		Short: "Render a complete sandbox spec from inputs (offline; no cluster interaction)",
		Long: `Render a complete sandbox spec from structured inputs.

render is a pure function: inputs -> complete sandbox YAML on stdout. It never
talks to the cluster; pipe its output into 'signadot sandbox apply -f -'.

Input modes:
  a) built-in template + flags/values:
       # single fork
       signadot sandbox render --cluster prod-eks --namespace hotrod \
         --fork route --image ghcr.io/acme/route:tag
       # multiple forks, shared image template
       signadot sandbox render --cluster prod-eks --namespace hotrod \
         --image-template ghcr.io/acme/{workload}:{sha} \
         --fork route --fork frontend
       # multiple forks, per-fork attrs (name[,image=...,namespace=...,kind=...])
       signadot sandbox render --cluster prod-eks \
         --fork route,namespace=hotrod,image=ghcr.io/acme/route:{sha} \
         --fork frontend,namespace=web,image=ghcr.io/acme/frontend:{sha}
       # from a values document
       signadot sandbox render --cluster prod-eks --values values.yaml
  b) user template file with @{var} placeholders:
       signadot sandbox render -f .signadot/tpl.yaml --set image=X
  c) a YAML merge patch (--patch) may be applied last in any mode.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return render(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	cfg.AddFlags(cmd)
	return cmd
}

func render(cfg *config.SandboxRender, out, log io.Writer) error {
	cctx, err := detectCIContext(cfg.Context)
	if err != nil {
		return err
	}

	var sb *models.Sandbox
	if cfg.Filename != "" {
		// Mode (b): user template file with @{var} placeholders.
		if cfg.Template != "" || cfg.ValuesFile != "" || len(cfg.Forks) > 0 || cfg.Image != "" || cfg.ImageTemplate != "" {
			return errors.New("-f/--filename is mutually exclusive with --template, --values, --fork, --image and --image-template")
		}
		sb, err = renderUserTemplate(cfg)
		if err != nil {
			return err
		}
	} else {
		// Mode (a): built-in template.
		if err := checkTemplateName(cfg.Template); err != nil {
			return err
		}
		if cfg.ValuesFile != "" && len(cfg.Forks) > 0 {
			return errors.New("--fork is mutually exclusive with --values")
		}
		vals, err := buildValues(cfg)
		if err != nil {
			return err
		}
		sb, err = compileForkDeployment(vals, cctx)
		if err != nil {
			return err
		}
	}
	if sb.Spec == nil {
		sb.Spec = &models.SandboxSpec{}
	}

	if err := applyContext(sb, cctx, cfg.Name, cfg.TTL); err != nil {
		return err
	}

	if cfg.Validate != "none" {
		if err := validateSandbox(sb); err != nil {
			return err
		}
	}

	// Convert to an unstructured {name, spec} document and apply --patch last.
	doc, err := sandboxToDoc(sb)
	if err != nil {
		return err
	}
	if cfg.PatchFile != "" {
		patch, err := loadPatch(cfg.PatchFile)
		if err != nil {
			return err
		}
		doc = mergePatch(doc, patch)
	}

	return writeRendered(cfg, out, doc)
}

// buildValues assembles a Values document for the built-in template, either from
// --values or from the single-fork sugar flags, folding in shared overrides.
func buildValues(cfg *config.SandboxRender) (*Values, error) {
	var vals *Values
	if cfg.ValuesFile != "" {
		data, err := readFileOrStdin(cfg.ValuesFile)
		if err != nil {
			return nil, err
		}
		vals, err = parseValues(data)
		if err != nil {
			return nil, err
		}
	} else {
		if len(cfg.Forks) == 0 {
			return nil, errors.New("nothing to render: specify --fork, --values, or -f")
		}
		if cfg.Image != "" && len(cfg.Forks) > 1 {
			return nil, errors.New("--image is only valid with a single --fork; use per-fork image=... or --image-template for multiple forks")
		}
		forks := make([]ValuesFork, 0, len(cfg.Forks))
		for _, spec := range cfg.Forks {
			f, err := parseForkSpec(spec)
			if err != nil {
				return nil, err
			}
			if f.Kind == "" {
				f.Kind = cfg.Kind
			}
			if cfg.Image != "" {
				if f.Image != "" {
					return nil, fmt.Errorf("fork %q: --image conflicts with the inline image= attribute", f.Workload)
				}
				f.Image = cfg.Image
			}
			forks = append(forks, f)
		}
		vals = &Values{Forks: forks}
	}

	// Structural overrides (name and ttl are applied later, with flag precedence).
	if cfg.Cluster != "" {
		vals.Cluster = cfg.Cluster
	}
	if cfg.Namespace != "" || cfg.ImageTemplate != "" {
		if vals.Defaults == nil {
			vals.Defaults = &ValuesDefaults{}
		}
		if cfg.Namespace != "" {
			vals.Defaults.Namespace = cfg.Namespace
		}
		if cfg.ImageTemplate != "" {
			vals.Defaults.ImageTemplate = cfg.ImageTemplate
		}
	}
	return vals, nil
}

func renderUserTemplate(cfg *config.SandboxRender) (*models.Sandbox, error) {
	sb, err := loadSandbox(cfg.Filename, cfg.TemplateVals, false)
	if err != nil {
		return nil, err
	}
	if sb.Spec == nil {
		sb.Spec = &models.SandboxSpec{}
	}
	if cfg.Cluster != "" {
		sb.Spec.Cluster = strPtr(cfg.Cluster)
	}
	return sb, nil
}

func checkTemplateName(t string) error {
	if t == "" {
		return nil // defaults to fork-deployment
	}
	base := t
	if i := strings.IndexByte(t, '@'); i >= 0 {
		base = t[:i]
	}
	if base != "fork-deployment" {
		return fmt.Errorf("unknown template %q (available: fork-deployment)", t)
	}
	return nil
}

// validateSandbox performs client-side render-time validation (§2.4).
func validateSandbox(sb *models.Sandbox) error {
	if sb.Name == "" {
		return errors.New("rendered sandbox has no name")
	}
	if len(sb.Name) > maxNameLen {
		return fmt.Errorf("sandbox name %q exceeds %d bytes", sb.Name, maxNameLen)
	}
	if sb.Spec == nil || sb.Spec.Cluster == nil || *sb.Spec.Cluster == "" {
		return errors.New("rendered sandbox spec must specify a cluster")
	}
	if len(sb.Spec.Forks) == 0 && len(sb.Spec.Local) == 0 {
		return errors.New("rendered sandbox spec must specify at least one fork or local workload")
	}
	for i, f := range sb.Spec.Forks {
		if f.ForkOf == nil || f.ForkOf.Name == nil || *f.ForkOf.Name == "" {
			return fmt.Errorf("forks[%d]: forkOf.name is required", i)
		}
	}
	return nil
}

// sandboxToDoc renders the sandbox to a {name, spec} map with null-valued keys
// pruned, so the emitted YAML is a clean, apply-ready manifest.
func sandboxToDoc(sb *models.Sandbox) (map[string]any, error) {
	specJSON, err := json.Marshal(sb.Spec)
	if err != nil {
		return nil, err
	}
	var spec map[string]any
	if err := json.Unmarshal(specJSON, &spec); err != nil {
		return nil, err
	}
	doc := map[string]any{
		"name": sb.Name,
		"spec": spec,
	}
	return pruneNulls(doc).(map[string]any), nil
}

func pruneNulls(v any) any {
	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			if val == nil {
				delete(x, k)
				continue
			}
			x[k] = pruneNulls(val)
		}
		return x
	case []any:
		for i := range x {
			x[i] = pruneNulls(x[i])
		}
		return x
	default:
		return v
	}
}

func loadPatch(path string) (map[string]any, error) {
	data, err := readFileOrStdin(path)
	if err != nil {
		return nil, err
	}
	patch := map[string]any{}
	if err := sigsyaml.Unmarshal(data, &patch); err != nil {
		return nil, fmt.Errorf("couldn't parse patch %q: %w", path, err)
	}
	return patch, nil
}

// mergePatch applies an RFC7386-style JSON merge patch (dependency-free).
func mergePatch(dst, patch map[string]any) map[string]any {
	for k, pv := range patch {
		if pv == nil {
			delete(dst, k)
			continue
		}
		if pm, ok := pv.(map[string]any); ok {
			if dm, ok := dst[k].(map[string]any); ok {
				dst[k] = mergePatch(dm, pm)
				continue
			}
		}
		dst[k] = pv
	}
	return dst
}

func writeRendered(cfg *config.SandboxRender, out io.Writer, doc map[string]any) error {
	switch cfg.OutputFormat {
	case config.OutputFormatJSON:
		return print.RawJSON(out, doc)
	case config.OutputFormatYAML, config.OutputFormatDefault:
		return print.RawK8SYAML(out, doc)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}

func readFileOrStdin(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}
