package override

import (
	"fmt"
	"io"
	"reflect"

	"github.com/signadot/cli/internal/builder"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
)

func createSandboxWithMiddleware(cfg *config.LocalOverrideCreate, baseSandbox *models.Sandbox, workloadName string, logPort int) (*models.Sandbox, string, func(io.Writer) error, error) {
	policyArg, err := builder.NewOverrideArgPolicy(cfg.ExcludedStatusCodes)
	if err != nil {
		return nil, "", noUnedit, err
	}
	var log *builder.MiddlewareOverrideArg
	if logPort > 0 {
		log, err = builder.NewOverrideLogArg(logPort)
		if err != nil {
			return nil, "", noUnedit, err
		}
	}
	sbBuilder := builder.
		BuildSandbox(cfg.Sandbox, builder.WithData(*baseSandbox)).
		AddOverrideMiddleware(cfg.Port, cfg.To, []string{workloadName}, policyArg, log).
		SetMachineID()

	sb, err := sbBuilder.Build()
	if err != nil {
		return nil, "", noUnedit, err
	}

	hasChanges := !reflect.DeepEqual(sb, baseSandbox)
	overrideName := *sbBuilder.GetLastAddedOverrideName()
	if !hasChanges {
		return &sb, overrideName, noUnedit, nil
	}

	sbParams := sandboxes.
		NewApplySandboxParams().
		WithOrgName(cfg.Org).
		WithSandboxName(cfg.Sandbox).
		WithData(&sb)

	resp, err := cfg.Client.Sandboxes.ApplySandbox(sbParams, nil)
	if err != nil {
		return nil, "", noUnedit, err
	}
	return resp.Payload, overrideName, mkUnedit(cfg, overrideName), nil
}

func deleteMiddlewareFromSandbox(cfg *config.LocalOverrideCreate, sandbox *models.Sandbox, overrideName string) error {
	sbBuilder := builder.
		BuildSandbox(cfg.Sandbox, builder.WithData(*sandbox)).
		SetMachineID().
		DeleteOverrideMiddleware(overrideName)

	sb, err := sbBuilder.Build()
	if err != nil {
		return err
	}

	if err := cfg.API.RefreshAPIConfig(); err != nil {
		return err
	}

	sbParams := sandboxes.
		NewApplySandboxParams().
		WithOrgName(cfg.Org).
		WithSandboxName(cfg.Sandbox).
		WithData(&sb)

	_, err = cfg.Client.Sandboxes.ApplySandbox(sbParams, nil)
	return err
}

func mkUnedit(cfg *config.LocalOverrideCreate, overrideName string) func(io.Writer) error {
	return func(out io.Writer) error {
		printOverrideProgress(out, fmt.Sprintf("Removing redirect in %s", cfg.Sandbox))
		sandbox, err := getSandbox(cfg)
		if err != nil {
			return err
		}
		return deleteMiddlewareFromSandbox(cfg, sandbox, overrideName)
	}
}

func noUnedit(io.Writer) error {
	return nil
}
