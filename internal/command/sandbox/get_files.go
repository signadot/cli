package sandbox

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/local"
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
			return getFiles(cfg, cmd.OutOrStdout(), args[0])
		},
	}
	cfg.AddFlags(cmd)

	return cmd
}

func getFiles(cfg *config.SandboxGetFiles, out io.Writer, name string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	// Get Sandbox
	params := sandboxes.NewGetSandboxParams().WithOrgName(cfg.Org).WithSandboxName(name)
	resp, err := cfg.Client.Sandboxes.GetSandbox(params, nil)
	if err != nil {
		return err
	}
	// get kube client
	kc, err := local.GetLocalKubeClient()
	if err != nil {
		return err
	}
	// extract
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	k8sEnv, sbLocal, err := extract(ctx, kc, resp.Payload, cfg.Local, cfg.Container)
	if err != nil {
		return err
	}
	err = calculateFileOverrides(ctx, kc, *sbLocal.From.Namespace, k8sEnv.Files, sbLocal.Files)
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
	// export
	// TODO: no-clobber
	if err := k8sEnv.Files.ExportTo(cfg.OutputDir); err != nil {
		return err
	}
	// print
	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		_, err := out.Write([]byte(cfg.OutputDir + "\n"))
		if err != nil {
			return err
		}

		return printTree(out, k8sEnv.Files, 0, k8sEnv.Files.IsDir() && len(k8sEnv.Files.Children) == 1)
	case config.OutputFormatJSON:
		return print.RawJSON(out, k8sEnv.Files)
	case config.OutputFormatYAML:
		return print.RawYAML(out, k8sEnv.Files)
	default:
		return fmt.Errorf("unknown output format %q", cfg.OutputFormat)
	}
}

func calculateFileOverrides(ctx context.Context, kubeClient client.Client, ns string, files *k8senv.Files, fileOps []*models.SandboxFiles) error {
	cmMap := map[string]*corev1.ConfigMap{}
	secMap := map[string]*corev1.Secret{}
	for _, fileOp := range fileOps {
		child := files.Path(fileOp.Path)
		if fileOp.ValueFrom == nil {
			child.Content = []byte(fileOp.Value)
			continue
		}
		if err := overrideValueFrom(ctx, kubeClient, child, fileOp, ns, cmMap, secMap); err != nil {
			return err
		}
	}
	return nil
}

func overrideValueFrom(ctx context.Context, kubeClient client.Client, child *k8senv.Files, fileOp *models.SandboxFiles, ns string, cmMap map[string]*corev1.ConfigMap, secMap map[string]*corev1.Secret) error {
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
	default:
		return fmt.Errorf("no definition for path %s", fileOp.Path)
	}
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
	turnStyle []byte = []byte("├")
	bar       []byte = []byte(strings.Repeat("─", 2))
	space     []byte = []byte(strings.Repeat(" ", 4))
	highL     []byte = []byte("└")
)

func printTree(out io.Writer, files *k8senv.Files, depth int, last bool) error {
	var err error
	for i := 0; i < depth; i++ {
		if i > 0 {
			_, err = out.Write(space)
			if err != nil {
				return err
			}
		}
		if i != depth-1 {
			continue
		}
		if last {
			_, err = out.Write(highL)
		} else {
			_, err = out.Write(turnStyle)
		}
		if err != nil {
			return err
		}
		_, err = out.Write(bar)
		if err != nil {
			return err
		}
		_, err = out.Write([]byte(" " + files.Name + "\n"))
	}
	if files.IsDir() {
		N := len(files.Children)
		n := 0
		for _, v := range files.Children {
			if v.IsDir() {
				continue
			}
			n++
			if err := printTree(out, v, depth+1, n == N); err != nil {
				return err
			}
		}
		for _, v := range files.Children {
			if !v.IsDir() {
				continue
			}
			n++
			if err := printTree(out, v, depth+1, n == N); err != nil {
				return err
			}
		}
		return nil
	}
	return err
}
