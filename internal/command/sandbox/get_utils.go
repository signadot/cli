package sandbox

import (
	"context"
	"errors"
	"fmt"

	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts"
	rolloutapi "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/signadot/cli/internal/locald/sandboxmanager"
	"github.com/signadot/go-sdk/models"
	"github.com/signadot/libconnect/common/k8senv"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func extract(ctx context.Context, kubeClient client.Client, sb *models.Sandbox, localName, containerName string) (*k8senv.ContainerEnv, *models.Local, error) {
	sbLocal, err := resolveLocal(sb, localName)
	if err != nil {
		return nil, nil, err
	}
	sel, err := resolveLabelSelector(ctx, kubeClient, sbLocal)
	if err != nil {
		return nil, nil, err
	}
	container, pod, err := resolveContainer(ctx, kubeClient, sel, *sbLocal.From.Namespace, containerName)
	if err != nil {
		return nil, nil, err
	}
	x := k8senv.NewExtractor(kubeClient)
	k8sEnv, err := x.ExtractContainer(ctx, container, pod)
	if err != nil {
		return nil, nil, err
	}
	return k8sEnv, sbLocal, nil
}

func extractSBEnvVar(ctx context.Context, kubeClient client.Client, ns string, resOuts []sandboxmanager.ResourceOutput, sbEnvVar *models.SandboxEnvVar) (*k8senv.EnvItem, error) {
	if sbEnvVar.ValueFrom == nil {
		return &k8senv.EnvItem{
			Name:  sbEnvVar.Name,
			Value: sbEnvVar.Value,
		}, nil
	}
	vf := sbEnvVar.ValueFrom
	switch {
	case vf.ConfigMap != nil:
		cm := &corev1.ConfigMap{}
		key := client.ObjectKey{Namespace: ns, Name: vf.ConfigMap.Name}
		err := kubeClient.Get(ctx, key, cm)
		if err != nil && vf.ConfigMap.Optional {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		val, ok := cm.Data[vf.ConfigMap.Key]
		if !ok && vf.ConfigMap.Optional {
			return nil, nil
		}
		if !ok {
			return nil, fmt.Errorf("key %q not present in ConfigMap %q", vf.ConfigMap.Key, vf.ConfigMap.Name)
		}
		return &k8senv.EnvItem{
			Name:  sbEnvVar.Name,
			Value: val,
		}, nil
	case vf.Secret != nil:
		secret := &corev1.Secret{}
		key := client.ObjectKey{Namespace: ns, Name: vf.Secret.Name}
		err := kubeClient.Get(ctx, key, secret)
		if err != nil && vf.Secret.Optional {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		val, ok := secret.Data[vf.Secret.Key]
		if !ok && vf.Secret.Optional {
			return nil, nil
		}
		if !ok {
			return nil, fmt.Errorf("key %q not present in Secret %q", vf.Secret.Key, vf.Secret.Name)
		}
		return &k8senv.EnvItem{
			Name:  sbEnvVar.Name,
			Value: string(val),
		}, nil
	case vf.Resource != nil:
		vfr := vf.Resource
		for i := range resOuts {
			out := &resOuts[i]
			if out.Resource != vfr.Name {
				continue
			}
			if out.Output != vfr.OutputKey {
				continue
			}
			return &k8senv.EnvItem{
				Name:  sbEnvVar.Name,
				Value: out.Value,
			}, nil
		}
		return nil, fmt.Errorf("output %q in resource %q unavailable", vfr.OutputKey, vfr.Name)
	default:
		return nil, fmt.Errorf("env var %q has no definition", sbEnvVar.Name)
	}
}

func resolveLocal(sb *models.Sandbox, localName string) (*models.Local, error) {
	locals := sb.Spec.Local
	if len(locals) == 0 {
		return nil, fmt.Errorf("no local in sandbox %s", sb.Name)
	}
	if localName == "" {
		return locals[0], nil
	}
	for i := range locals {
		local := locals[i]
		if local.Name == localName {
			return local, nil
		}
	}
	return nil, fmt.Errorf("local %s not found in sandbox %s", localName, sb.Name)
}

func resolveLabelSelector(ctx context.Context, kubeClient client.Client, sbLocal *models.Local) (*metav1.LabelSelector, error) {
	from := sbLocal.From
	if from == nil {
		return nil, fmt.Errorf("no baseline specified in local %s", sbLocal.Name)
	}
	switch *from.Kind {
	case rollouts.RolloutKind:
		r := &rolloutapi.Rollout{}
		err := kubeClient.Get(ctx, client.ObjectKey{
			Namespace: *from.Namespace,
			Name:      *from.Name,
		}, r)
		if err != nil {
			return nil, err
		}
		return r.Spec.Selector, nil
	case "Deployment":
		d := &appsv1.Deployment{}
		err := kubeClient.Get(ctx, client.ObjectKey{
			Namespace: *from.Namespace,
			Name:      *from.Name,
		}, d)
		if err != nil {
			return nil, err
		}
		return d.Spec.Selector, nil
	default:
		return nil, fmt.Errorf("unknown kind %q (expected %s or %s)", sbLocal.From.Kind, "Deployment", rollouts.RolloutKind)

	}
}

func resolveContainer(ctx context.Context, kubeClient client.Client, sel *metav1.LabelSelector, namespace, containerName string) (*corev1.Container, *corev1.Pod, error) {
	selector, err := metav1.LabelSelectorAsSelector(sel)
	if err != nil {
		return nil, nil, err
	}
	listOpts := &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: selector,
	}
	podList := &corev1.PodList{}
	if err := kubeClient.List(ctx, podList, listOpts); err != nil {
		return nil, nil, err
	}
	if len(podList.Items) == 0 {
		return nil, nil, errors.New("no pods found")
	}
	pod := &podList.Items[0]
	for i := range pod.Spec.Containers {
		c := &pod.Spec.Containers[i]
		if c.Name == containerName || containerName == "" {
			return c, pod, nil
		}
	}
	return nil, nil, fmt.Errorf("no container %q in selected pods", containerName)
}
