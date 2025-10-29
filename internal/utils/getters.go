package utils

import (
	"context"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
)

func GetSandbox(ctx context.Context, cfg *config.API,
	sandboxName string) (*models.Sandbox, error) {
	// make sure the auth token is refreshed
	if err := cfg.RefreshAPIConfig(); err != nil {
		return nil, err
	}

	// get the sandbox
	sandboxParams := sandboxes.NewGetSandboxParams().
		WithContext(ctx).WithOrgName(cfg.Org).WithSandboxName(sandboxName)
	resp, err := cfg.Client.Sandboxes.GetSandbox(sandboxParams, nil)
	if err != nil {
		return nil, err
	}
	return resp.Payload, nil
}
