package sandbox

import (
	"context"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/local"
	"github.com/signadot/cli/internal/locald/sandboxmanager"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
	"github.com/signadot/libconnect/common/k8senv"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newGetEnv(sandbox *config.Sandbox) *cobra.Command {
	cfg := &config.SandboxGetEnv{Sandbox: sandbox}

	cmd := &cobra.Command{
		Use:   "get-env NAME",
		Short: "Get environment from a (local) sandbox",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return getEnv(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args[0])
		},
	}
	cfg.AddFlags(cmd)

	return cmd
}

func getEnv(cfg *config.SandboxGetEnv, out, errOut io.Writer, name string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	// Get Sandbox
	params := sandboxes.NewGetSandboxParams().WithOrgName(cfg.Org).WithSandboxName(name)
	resp, err := cfg.Client.Sandboxes.GetSandbox(params, nil)
	if err != nil {
		return err
	}
	apiSB := resp.Payload
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// get resource outputs if needed
	var resourceOutputs []sandboxmanager.ResourceOutput
	if hasEnvResourceRefs(apiSB) {
		resourceOutputs, err = sandboxmanager.GetResourceOutputs(ctx, apiSB.RoutingKey)
		if err != nil {
			return err
		}
	}
	// get kube client
	kc, err := local.GetLocalKubeClient()
	if err != nil {
		return err
	}
	// extract
	k8sEnv, sbLocal, err := extract(ctx, kc, apiSB, cfg.Local, cfg.Container)
	if err != nil {
		return err
	}
	// overrides
	resEnv, err := calculateOverrides(ctx, kc, *sbLocal.From.Namespace, resourceOutputs, k8sEnv.Env, sbLocal.Env)
	if err != nil {
		return err
	}
	// print errors
	if err := printForbidden(errOut, k8sEnv.Forbidden); err != nil {
		return err
	}
	// print output
	return printEnv(out, cfg.OutputFormat, resEnv)
}

func printEnv(out io.Writer, oFmt config.OutputFormat, resEnv []k8senv.EnvItem) error {
	switch oFmt {
	case config.OutputFormatDefault:
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 1, ' ', tabwriter.TabIndent)
		for _, item := range resEnv {
			_, err := w.Write([]byte(item.ToShellEval() + "\n"))
			if err != nil {
				return err
			}
		}
		return w.Flush()
	case config.OutputFormatJSON:
		return print.RawJSON(out, resEnv)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resEnv)
	default:
		return fmt.Errorf("unknown output format %q", oFmt)
	}
}

func calculateOverrides(ctx context.Context, kubeClient client.Client, ns string, resOuts []sandboxmanager.ResourceOutput, xEnv []k8senv.EnvItem, sbEnv []*models.SandboxEnvVar) ([]k8senv.EnvItem, error) {
	sbEnvMap := map[string]*k8senv.EnvItem{}
	for _, sbEnvVar := range sbEnv {
		xEnvVar, err := extractSBEnvVar(ctx, kubeClient, ns, resOuts, sbEnvVar)
		if err != nil {
			return nil, err
		}
		if xEnvVar == nil {
			continue
		}
		sbEnvMap[xEnvVar.Name] = xEnvVar
	}
	for i := range xEnv {
		xEnvVar := &xEnv[i]
		sbEnvVar, ok := sbEnvMap[xEnvVar.Name]
		if !ok {
			continue
		}
		xEnvVar.Value = sbEnvVar.Value
		xEnvVar.Source = sbEnvVar.Source
		delete(sbEnvMap, xEnvVar.Name)
	}
	res := xEnv
	for _, v := range sbEnvMap {
		res = append(res, *v)
	}
	return res, nil
}

func hasEnvResourceRefs(apiSB *models.Sandbox) bool {
	for _, local := range apiSB.Spec.Local {
		for _, env := range local.Env {
			if env.ValueFrom == nil {
				continue
			}
			vf := env.ValueFrom
			if vf.Resource == nil {
				continue
			}
			return true
		}
	}
	return false
}
