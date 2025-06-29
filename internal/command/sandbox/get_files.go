package sandbox

import (
	"context"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/local"
	"github.com/signadot/cli/internal/locald/sandboxmanager"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
	"github.com/signadot/libconnect/common/k8senv"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newGetFiles(sandbox *config.Sandbox) *cobra.Command {
	cfg := &config.SandboxGetFiles{Sandbox: sandbox}

	cmd := &cobra.Command{
		Use:   "get-files NAME",
		Short: "Get files from a (local) sandbox",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return getFiles(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args[0])
		},
	}
	cfg.AddFlags(cmd)

	return cmd
}

func getFiles(cfg *config.SandboxGetFiles, out, errOut io.Writer, name string) error {
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
	// get kube client
	kc, err := local.GetLocalKubeClient()
	if err != nil {
		return err
	}
	// extract k8senv
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	k8sEnv, sbLocal, err := extract(ctx, kc, apiSB, cfg.Local, cfg.Container)
	if err != nil {
		return err
	}

	var resourceOutputs []sandboxmanager.ResourceOutput
	if hasFileResourceOutput(apiSB) {
		resourceOutputs, err = sandboxmanager.GetResourceOutputs(ctx, apiSB.RoutingKey)
		if err != nil {
			return err
		}
	}
	err = calculateFileOverrides(ctx, kc, *sbLocal.From.Namespace, resourceOutputs, k8sEnv.Files, sbLocal.Files)
	if err != nil {
		return err
	}
	absOut, err := filepath.Abs(cfg.OutputDir)
	if err != nil {
		return err
	}
	err = writeGCFiles(k8sEnv.Files, absOut)
	if err != nil {
		return err
	}
	// no-clobber
	if cfg.NoClobber {
		if _, err := noClobber(errOut, k8sEnv.Files, cfg.OutputDir); err != nil {
			return err
		}
	}
	// export
	if err := k8sEnv.Files.ExportTo(cfg.OutputDir); err != nil {
		return err
	}
	// print
	if err := printForbidden(errOut, k8sEnv.Forbidden); err != nil {
		return err
	}
	k8sEnv.Files.Name = cfg.OutputDir
	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return printTree(out, k8sEnv.Files, []bool{})
	case config.OutputFormatJSON:
		return print.RawJSON(out, k8sEnv.Files)
	case config.OutputFormatYAML:
		return print.RawYAML(out, k8sEnv.Files)
	default:
		return fmt.Errorf("unknown output format %q", cfg.OutputFormat)
	}
}

func calculateFileOverrides(ctx context.Context, kubeClient client.Client, ns string, resOuts []sandboxmanager.ResourceOutput, files *k8senv.Files, fileOps []*models.SandboxFileOp) error {
	cmMap := map[string]*corev1.ConfigMap{}
	secMap := map[string]*corev1.Secret{}

	for _, fileOp := range fileOps {
		child := files.Path(fileOp.Path)
		if fileOp.ValueFrom == nil {
			child.Content = []byte(fileOp.Value)
			continue
		}
		if err := overrideFileValueFrom(ctx, kubeClient, child, fileOp, ns, resOuts, cmMap, secMap); err != nil {
			return err
		}
	}
	return nil
}

func overrideFileValueFrom(ctx context.Context, kubeClient client.Client, child *k8senv.Files, fileOp *models.SandboxFileOp, ns string, resOuts []sandboxmanager.ResourceOutput, cmMap map[string]*corev1.ConfigMap, secMap map[string]*corev1.Secret) error {
	vf := fileOp.ValueFrom
	switch {
	case vf.ConfigMap != nil:
		cmSource := vf.ConfigMap
		cm := cmMap[cmSource.Name]
		if cm == nil {
			cm = &corev1.ConfigMap{}
			err := kubeClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: cmSource.Name}, cm)
			if cmSource.Optional && k8serrors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
			cmMap[cmSource.Name] = cm
		}
		if cmSource.Key == "" {
			child.MountConfigMap(cm)
		} else {
			val, ok := cm.Data[cmSource.Key]
			if !cmSource.Optional && !ok {
				return fmt.Errorf("key %q not found in configMap %q", cmSource.Key, cmSource.Name)
			}
			if !ok {
				return nil
			}
			child.Content = []byte(val)
		}
		return nil

	case vf.Secret != nil:
		cmSource := vf.Secret
		cm := secMap[cmSource.Name]
		if cm == nil {
			cm = &corev1.Secret{}
			err := kubeClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: cmSource.Name}, cm)
			if cmSource.Optional && k8serrors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
			secMap[cmSource.Name] = cm
		}
		if cmSource.Key == "" {
			child.MountSecret(cm)
		} else {
			val, ok := cm.Data[cmSource.Key]
			if !cmSource.Optional && !ok {
				return fmt.Errorf("key %q not found in secret %q", cmSource.Key, cmSource.Name)
			}
			if !ok {
				return nil
			}
			child.Content = val
		}
		return nil
	case vf.Resource != nil:
		vfr := vf.Resource
		for i := range resOuts {
			resOut := &resOuts[i]
			if vfr.Name != resOut.Resource {
				continue
			}
			if vfr.OutputKey == resOut.Output {
				child.Content = []byte(resOut.Value)
				return nil
			}
			if vfr.OutputKey != "" {
				continue
			}
			// all resource outputs mounted.
			keyChild := child.Path(resOut.Output)
			keyChild.Content = []byte(resOut.Value)
		}
		return nil

	default:
		return fmt.Errorf("no definition for path %s: %#v", fileOp.Path, vf)
	}
}

func noClobber(errOut io.Writer, files *k8senv.Files, base string) (bool, error) {
	if files.IsDir() {
		for k, v := range files.Children {
			p := filepath.Join(base, v.Name)
			remove, err := noClobber(errOut, v, p)
			if err != nil {
				return false, err
			}
			if remove {
				delete(files.Children, k)
			}
		}
		return false, nil
	}
	_, err := os.Stat(base)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	fmt.Fprintf(errOut, "%s already exists, skipping\n", base)
	return true, nil
}

func writeGCFiles(files *k8senv.Files, base string) error {
	if files.IsDir() {
		for k, child := range files.Children {
			if err := writeGCFiles(child, filepath.Join(base, k)); err != nil {
				return err
			}
		}
		return nil
	}
	return utils.RegisterPathForGC(base)
}

var (
	vBar      []byte = []byte("│   ")
	highL     []byte = []byte("└── ")
	turnStyle []byte = []byte("├── ")
	space     []byte = []byte(strings.Repeat(" ", 4))
)

func printTree(out io.Writer, files *k8senv.Files, ended []bool) error {
	var err error
	for i, b := range ended {
		if i < len(ended)-1 {
			if !b {
				_, err = out.Write(vBar)
			} else {
				_, err = out.Write(space)
			}
		} else if b {
			_, err = out.Write(highL)
		} else {
			_, err = out.Write(turnStyle)
		}
		if err != nil {
			return err
		}
	}
	_, err = out.Write([]byte(files.Name + "\n"))
	if err != nil {
		return err
	}
	if files.IsDir() {
		N := len(files.Children)
		n := 0
		keys := slices.Sorted(maps.Keys(files.Children))
		for _, k := range keys {
			n++
			if err := printTree(out, files.Children[k], append(ended, n == N)); err != nil {
				return err
			}
		}
	}
	return nil
}

func hasFileResourceOutput(sb *models.Sandbox) bool {
	for _, local := range sb.Spec.Local {
		for _, fileOp := range local.Files {
			if fileOp.ValueFrom == nil {
				continue
			}
			if fileOp.ValueFrom.Resource == nil {
				continue
			}
			return true
		}
	}
	return false
}
