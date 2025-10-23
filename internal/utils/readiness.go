package utils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/poll"
	"github.com/signadot/cli/internal/spinner"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
)

func WaitForSandboxReady(ctx context.Context, cfg *config.API, out io.Writer,
	sandboxName string, waitTimeout time.Duration) (*models.Sandbox, error) {
	fmt.Fprintf(out, "Waiting (up to --wait-timeout=%v) for sandbox to be ready...\n", waitTimeout)

	params := sandboxes.NewGetSandboxParams().
		WithContext(ctx).
		WithOrgName(cfg.Org).
		WithSandboxName(sandboxName)

	spin := spinner.Start(out, "Sandbox status")
	defer spin.Stop()

	var sb *models.Sandbox

	retry := poll.
		NewPoll().
		WithTimeout(waitTimeout)
	var failedErr error
	err := retry.Until(ctx, func(ctx context.Context) bool {
		result, err := cfg.Client.Sandboxes.GetSandbox(params, nil)
		if err != nil {
			// Keep retrying in case it's a transient error.
			spin.Messagef("error: %v", err)
			return false
		}
		sb = result.Payload
		if !sb.Status.Ready {
			if sb.Status.Reason == "ResourceFailed" {
				failedErr = errors.New(sb.Status.Message)
				return true
			}
			spin.Messagef("Not Ready: %s", sb.Status.Message)
			return false
		}
		spin.StopMessagef("Ready: %s", sb.Status.Message)
		return true
	})

	if failedErr != nil {
		spin.StopFail()
		return sb, failedErr
	}

	if err != nil {
		spin.StopFail()
		return sb, err
	}

	return sb, nil
}
