package override

import (
	"context"
	"fmt"
	"io"
	"reflect"

	"github.com/signadot/cli/internal/builder"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/devbox"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
)

type undoFunc func(ctx context.Context, w io.Writer) error

func hasSandboxOverride(sb *models.Sandbox) bool {
	return len(builder.GetAvailableOverrideMiddlewares(sb)) > 0
}

func applyOverrideToSandbox(ctx context.Context, cfg *config.LocalOverrideCreate,
	baseSandbox *models.Sandbox, workloadName, devboxID string, logPort int,
) (*models.Sandbox, string, undoFunc, error) {
	// generate the override mw policy arg
	policyArg, err := builder.NewOverrideArgPolicy(cfg.ExcludedStatusCodes)
	if err != nil {
		return nil, "", noOpUndo, err
	}

	// generate the override log arg
	var log *builder.MiddlewareOverrideArg
	if logPort > 0 {
		log, err = builder.NewOverrideLogArg(logPort)
		if err != nil {
			return nil, "", noOpUndo, err
		}
	}

	// build the sandbox
	sbBuilder := builder.
		BuildSandbox(cfg.Sandbox, builder.WithData(*baseSandbox)).
		AddOverrideMiddleware(cfg.Port, cfg.To, []string{workloadName}, policyArg, log).
		SetDevboxID(devboxID)

	sb, err := sbBuilder.Build()
	if err != nil {
		return nil, "", noOpUndo, err
	}

	// check if anything has changed
	hasChanges := !reflect.DeepEqual(sb, baseSandbox)
	overrideName := *sbBuilder.GetLastAddedOverrideName()
	if !hasChanges {
		return &sb, overrideName, noOpUndo, nil
	}

	// apply the sandbox
	sbParams := sandboxes.
		NewApplySandboxParams().
		WithContext(ctx).
		WithOrgName(cfg.Org).
		WithSandboxName(cfg.Sandbox).
		WithData(&sb)

	resp, err := cfg.Client.Sandboxes.ApplySandbox(sbParams, nil)
	if err != nil {
		return nil, "", noOpUndo, err
	}
	return resp.Payload, overrideName, mkUndo(cfg, overrideName), nil
}

func deleteOverrideFromSandbox(ctx context.Context, cfg *config.API,
	sandbox *models.Sandbox, overrideName string) error {
	// Use the sandbox builder to delete the override
	devboxID, err := devbox.GetID(ctx, cfg, false, "")
	if err != nil {
		return err
	}
	sbBuilder := builder.
		BuildSandbox(sandbox.Name, builder.WithData(*sandbox)).
		SetDevboxID(devboxID).
		DeleteOverrideMiddleware(overrideName)

	sb, err := sbBuilder.Build()
	if err != nil {
		return err
	}

	// Apply the updated sandbox
	params := sandboxes.NewApplySandboxParams().
		WithContext(ctx).
		WithOrgName(cfg.Org).
		WithSandboxName(sandbox.Name).
		WithData(&sb)

	_, err = cfg.Client.Sandboxes.ApplySandbox(params, nil)
	return err
}

func mkUndo(cfg *config.LocalOverrideCreate, overrideName string) undoFunc {
	return func(ctx context.Context, out io.Writer) error {
		sb, err := utils.GetSandbox(ctx, cfg.API, cfg.Sandbox)
		if err != nil {
			return err
		}
		printOverrideProgress(out, fmt.Sprintf("Removing override from %s", cfg.Sandbox))
		return deleteOverrideFromSandbox(ctx, cfg.API, sb, overrideName)
	}
}

func noOpUndo(context.Context, io.Writer) error {
	return nil
}
