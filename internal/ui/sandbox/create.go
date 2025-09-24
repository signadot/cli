package sandbox_ui

import (
	"context"
	"fmt"
	"time"

	"github.com/signadot/cli/internal/config"

	"github.com/charmbracelet/huh"
)

type SandboxCreateForm struct {
	formData *config.SandboxCreate
}

func Run(ctx context.Context, cfg *config.SandboxCreate) error {

	s := SandboxCreateForm{
		formData: cfg,
	}

	if cfg.NoInteractive {
		return nil
	}

	if cfg.Cluster == "" {
		err := s.getClusterInput().Run()
		if err != nil {
			return err
		}
	}

	if cfg.KubernetesWorkload == "" {
		err := s.getKubernetesWorkloadInput()
		if err != nil {
			return err
		}
	}

	if cfg.TTL == "" {
		err := s.getTTLInput()
		if err != nil {
			return err
		}
	}

	return nil
}

func getClusterOptions() []huh.Option[string] {
	return []huh.Option[string]{
		{Value: "signadot-staging", Key: "signadot-staging"},
		{Value: "signadot-production", Key: "signadot-production"},
		{Value: "demo", Key: "demo"},
	}
}

func (s *SandboxCreateForm) getClusterInput() *huh.Select[string] {
	return huh.
		NewSelect[string]().
		Title("Cluster").
		Description("The cluster to create the sandbox in").
		Value(&s.formData.Cluster).
		Options(getClusterOptions()...)

}

func (s *SandboxCreateForm) getKubernetesWorkloadInput() error {

	type workload struct {
		Kind      string
		Namespace string
		Name      string
	}

	w := workload{}

	kindRollout := huh.NewOption("Rollout", "Rollout")
	kindDeployment := huh.NewOption("Deployment", "Deployment")

	kind := huh.NewSelect[string]().
		Title("Kind").
		Description("The kind of the Kubernetes workload").
		Value(&w.Kind).
		Options(kindRollout, kindDeployment)

	if err := kind.Run(); err != nil {
		return err
	}

	n1 := huh.NewOption("default", "default")
	n2 := huh.NewOption("hotrod", "hotrod")
	n3 := huh.NewOption("hotrod-devmesh", "hotrod-devmesh")

	namespace := huh.NewSelect[string]().
		Title("Namespace").
		Description("The namespace of the Kubernetes workload").
		Value(&w.Namespace).
		Options(n1, n2, n3)

	if err := namespace.Run(); err != nil {
		return err
	}

	name := huh.NewInput().
		Title("Name").
		Description("The name of the Kubernetes workload").
		Value(&w.Name).
		SuggestionsFunc(func() []string {
			return []string{w.Kind, w.Namespace}
		}, nil)

	if err := name.Run(); err != nil {
		return err
	}

	s.formData.KubernetesWorkload = fmt.Sprintf("%s/%s/%s", w.Kind, w.Namespace, w.Name)

	return nil
}

func (s *SandboxCreateForm) getTTLInput() error {
	// First ask if user wants to add TTL
	wantsTTL := false
	confirm := huh.NewConfirm().
		Title("Add TTL?").
		Description("Do you want to set a time-to-live for this sandbox?").
		Value(&wantsTTL)

	if err := confirm.Run(); err != nil {
		return err
	}

	if !wantsTTL {
		// User doesn't want TTL, leave it empty
		return nil
	}

	// User wants TTL, ask for the duration
	ttlInput := huh.NewInput().
		Title("TTL Duration").
		Description("Enter the TTL duration (e.g., 1h, 30m, 2d, 1w)").
		Value(&s.formData.TTL).
		Validate(func(str string) error {
			if str == "" {
				return fmt.Errorf("TTL duration cannot be empty")
			}
			// Validate that it's a valid Go duration
			_, err := time.ParseDuration(str)
			if err != nil {
				// Try parsing with day and week units (not supported by time.ParseDuration)
				// but commonly used in TTL contexts
				if str == "1d" || str == "2d" || str == "3d" || str == "7d" ||
					str == "1w" || str == "2w" || str == "3w" || str == "4w" {
					return nil
				}
				return fmt.Errorf("invalid duration format: %v. Use formats like 1h, 30m, 2d, 1w", err)
			}
			return nil
		})

	return ttlInput.Run()
}
